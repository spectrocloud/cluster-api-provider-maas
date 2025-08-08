package machine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
	"k8s.io/klog/v2/textlogger"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Service manages the MaaS machine
var (
	ErrBrokenMachine = errors.New("broken machine encountered")
	reHostID         = regexp.MustCompile(`host (\d+)`)
	reMachineID      = regexp.MustCompile(`machine[s]? ([a-z0-9]{4,6})`)
)

type Service struct {
	scope      *scope.MachineScope
	maasClient maasclient.ClientSetInterface
}

// DNS service returns a new helper for managing a MaaS "DNS" (DNS client loadbalancing)
func NewService(machineScope *scope.MachineScope) *Service {
	return &Service{
		scope:      machineScope,
		maasClient: scope.NewMaasClient(machineScope.ClusterScope),
	}
}

// logVMHostDiagnostics attempts to extract the VM host (pod) id from a MAAS error message
// and, if found, fetches its details and prints status information to the controller log.
func logVMHostDiagnostics(s *Service, err error) {
	// First, check for machine id in the error and force-release it if found
	if m := reMachineID.FindStringSubmatch(err.Error()); len(m) == 2 {
		sys := m[1]
		s.scope.Info("Releasing broken machine", "system-id", sys)
		ctx := context.TODO()
		_, _ = s.maasClient.Machines().Machine(sys).Releaser().WithForce().Release(ctx)
	}

	matches := reHostID.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return // no host id in message
	}
	podID := matches[1]
	s.scope.Info("Broken VM host detected", "pod-id", podID)
	ctx := context.TODO()
	if vmHost, e := s.maasClient.VMHosts().VMHost(podID).Get(ctx); e == nil {
		s.scope.Info("VM host status", "pod-id", podID, "name", vmHost.Name(), "status", vmHost.Type(), "availCores", vmHost.AvailableCores(), "availMem", vmHost.AvailableMemory())
	} else {
		s.scope.Error(e, "failed to fetch VM host details", "pod-id", podID)
	}
}

func (s *Service) GetMachine(systemID string) (*infrav1beta1.Machine, error) {

	if systemID == "" {
		return nil, nil
	}

	m, err := s.maasClient.Machines().Machine(systemID).Get(context.Background())
	if err != nil {
		return nil, err
	}

	machine := fromSDKTypeToMachine(m)

	return machine, nil
}

func (s *Service) ReleaseMachine(systemID string) error {
	ctx := context.TODO()

	_, err := s.maasClient.Machines().
		Machine(systemID).
		Releaser().
		Release(ctx)
	if err != nil {
		return errors.Wrapf(err, "Unable to release machine")
	}

	return nil
}

func (s *Service) DeployMachine(userDataB64 string) (_ *infrav1beta1.Machine, rerr error) {
	ctx := context.TODO()
	log := textlogger.NewLogger(textlogger.NewConfig())

	mm := s.scope.MaasMachine

	// Check if we should create an LXD VM (Stage 1 or explicit flag)
	if s.scope.GetDynamicLXD() {
		s.scope.Info("Using LXD VM creation path", "machine", mm.Name)
		return s.createLXDVM(userDataB64)
	}

	// Standard MAAS machine allocation path
	s.scope.Info("Using standard MAAS machine allocation path", "machine", mm.Name)

	failureDomain := mm.Spec.FailureDomain
	if failureDomain == nil {
		if s.scope.Machine.Spec.FailureDomain != nil && *s.scope.Machine.Spec.FailureDomain != "" {
			failureDomain = s.scope.Machine.Spec.FailureDomain
		}
	}

	var m maasclient.Machine
	var err error

	if s.scope.GetProviderID() == "" {
		allocator := s.maasClient.
			Machines().
			Allocator().
			WithCPUCount(*mm.Spec.MinCPU).
			WithMemory(*mm.Spec.MinMemoryInMB)

		if failureDomain != nil {
			allocator.WithZone(*failureDomain)
		}

		if mm.Spec.ResourcePool != nil {
			allocator.WithResourcePool(*mm.Spec.ResourcePool)
		}

		if len(mm.Spec.Tags) > 0 {
			allocator.WithTags(mm.Spec.Tags)
		}

		s.scope.Info("Requesting MAAS allocation")
		m, err = allocator.Allocate(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "Invalid transition: Broken") {
				logVMHostDiagnostics(s, err)
				s.scope.Info("Broken machine encountered; will retry")
				return nil, ErrBrokenMachine
			}
			return nil, errors.Wrapf(err, "Unable to allocate machine")
		}

		s.scope.SetProviderID(m.SystemID(), m.Zone().Name())
		err = s.scope.PatchObject()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to pathc machine with provider id")
		}
	} else {
		m, err = s.maasClient.Machines().Machine(*s.scope.GetInstanceID()).Get(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find machine %s", *s.scope.GetInstanceID())
		}
	}

	s.scope.Info("Allocated machine", "system-id", m.SystemID())

	defer func() {
		if rerr != nil {
			s.scope.Info("Attempting to release machine which failed to deploy")
			_, err := m.Releaser().Release(ctx)
			if err != nil {
				// Is it right to NOT set rerr so we can see the original issue?
				log.Error(err, "Unable to release properly")
			}
		}
	}()

	// TODO need to revisit if we need to set the hostname OR not
	//Hostname: &mm.Name,
	noSwap := 0
	if _, err := m.Modifier().SetSwapSize(noSwap).Update(ctx); err != nil {
		return nil, errors.Wrapf(err, "Unable to disable swap")
	}

	s.scope.Info("Swap disabled", "system-id", m.SystemID())

	s.scope.Info("Starting deployment", "system-id", m.SystemID())
	deployingM, err := m.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).Deploy(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to deploy machine")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

// createLXDVM creates a new LXD VM and registers it with MAAS
// This method uses MAAS API for cross-cluster communication
func (s *Service) createLXDVM(userDataB64 string) (*infrav1beta1.Machine, error) {
	mm := s.scope.MaasMachine
	machineName := s.scope.Machine.Name

	// Generate a unique VM name
	vmName := fmt.Sprintf("lxd-vm-%s", machineName)
	// Record the intended VM name on the MaasMachine for observability
	if mm.Annotations == nil {
		mm.Annotations = map[string]string{}
	}
	mm.Annotations["maas.spectrocloud.com/vm-name"] = vmName
	_ = s.scope.PatchObject()

	s.scope.Info("Creating LXD VM via MAAS API", "name", vmName)

	// For workload clusters, we need to find LXD hosts and create VMs on them
	if s.scope.ClusterScope.IsWorkloadCluster() {
		return s.createLXDVMForWorkloadCluster(userDataB64)
	}

	// For infrastructure clusters, use the existing approach
	// This is a simplified implementation that needs to be enhanced
	s.scope.Info("Creating LXD VM on infrastructure cluster", "name", vmName)
	// request allocation
	s.scope.Info("Requesting MAAS allocation", "vm-name", vmName)

	// Use MAAS API to allocate a machine (which will be a VM on LXD host)
	allocator := s.maasClient.
		Machines().
		Allocator().
		WithCPUCount(*mm.Spec.MinCPU).
		WithMemory(*mm.Spec.MinMemoryInMB)

	// Apply placement filters and tags
	if mm.Spec.FailureDomain != nil {
		allocator = allocator.WithZone(*mm.Spec.FailureDomain)
	}
	if mm.Spec.ResourcePool != nil {
		allocator = allocator.WithResourcePool(*mm.Spec.ResourcePool)
	}

	// Add tags only when user supplied
	if len(mm.Spec.Tags) > 0 {
		allocator = allocator.WithTags(mm.Spec.Tags)
	}

	// Allocate the machine (MAAS will handle VM creation on LXD host)
	m, err := allocator.Allocate(context.TODO())
	if err != nil {
		s.scope.Error(err, "allocate failed")
		logVMHostDiagnostics(s, err)
		return nil, errors.Wrapf(err, "failed to allocate LXD VM via MAAS API")
	}

	// Set desired hostname before deployment
	if _, err := m.Modifier().SetHostname(vmName).Update(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "failed to set hostname before deploy")
	}

	// Record allocation success
	s.scope.Info("LXD VM allocated", "system-id", m.SystemID())
	// Start deployment now
	s.scope.Info("Starting deployment", "system-id", m.SystemID())

	// Set static IP if specified
	if staticIP := s.scope.GetStaticIP(); staticIP != "" {
		if err := s.configureStaticIPForMachine(m, staticIP); err != nil {
			s.scope.Error(err, "failed to configure static IP", "ip", staticIP)
			// Don't fail the entire operation
		}
	}

	// Deploy the VM with user data
	deployingM, err := m.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).Deploy(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deploy LXD VM")
	}

	// Set provider ID and system ID
	s.scope.SetProviderID(deployingM.SystemID(), deployingM.Zone().Name())
	if err := s.scope.PatchObject(); err != nil {
		return nil, errors.Wrapf(err, "failed to patch machine with provider ID")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

// configureStaticIPForMachine configures static IP for a machine
func (s *Service) configureStaticIPForMachine(m maasclient.Machine, staticIP string) error {
	// Simplified implementation - in real implementation, use proper MAAS API calls
	s.scope.Info("Configuring static IP", "ip", staticIP, "system-id", m.SystemID())

	// For now, just log the intent - actual implementation would use MAAS API
	// to configure the interface with static IP
	return nil
}

// shouldUseLXDForWorkloadCluster determines if a workload cluster machine should use LXD
func (s *Service) shouldUseLXDForWorkloadCluster() bool {
	// Check if this is a workload cluster
	if !s.scope.ClusterScope.IsWorkloadCluster() {
		return false
	}

	// Check if the infrastructure cluster has LXD enabled
	infraCluster, err := s.scope.ClusterScope.GetInfrastructureCluster()
	if err != nil {
		s.scope.Error(err, "failed to get infrastructure cluster")
		return false
	}

	if infraCluster.Spec.LXDControlPlaneCluster == nil || !*infraCluster.Spec.LXDControlPlaneCluster {
		s.scope.Info("Infrastructure cluster does not have LXD enabled")
		return false
	}

	// Check node pool configuration
	return s.scope.ClusterScope.ShouldUseLXDForMachine(s.scope.Machine)
}

// createLXDVMForWorkloadCluster creates an LXD VM for a workload cluster machine
func (s *Service) createLXDVMForWorkloadCluster(userDataB64 string) (*infrav1beta1.Machine, error) {
	mm := s.scope.MaasMachine
	machineName := s.scope.Machine.Name

	s.scope.Info("Creating LXD VM for workload cluster", "machine", machineName, "cluster", s.scope.Cluster.Name)

	// Get infrastructure cluster for LXD host selection
	_, err := s.scope.ClusterScope.GetInfrastructureCluster()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get infrastructure cluster")
	}

	// Get node pool configuration
	poolConfig := s.scope.ClusterScope.GetNodePoolConfig(s.scope.Machine)
	if poolConfig == nil {
		return nil, errors.New("no node pool configuration found for machine")
	}

	// Determine availability zone and resource pool
	availabilityZone := s.getAvailabilityZoneForWorkloadMachine(poolConfig)
	resourcePool := s.getResourcePoolForWorkloadMachine(poolConfig)

	// Get static IP if configured
	staticIP := s.scope.ClusterScope.GetStaticIPForMachine(s.scope.Machine)

	// Generate unique VM name
	vmName := fmt.Sprintf("workload-vm-%s-%s", s.scope.Cluster.Name, machineName)
	// Record the intended VM name on the MaasMachine so operators can trace it in MAAS
	if mm.Annotations == nil {
		mm.Annotations = map[string]string{}
	}
	mm.Annotations["maas.spectrocloud.com/vm-name"] = vmName
	_ = s.scope.PatchObject()

	s.scope.Info("Creating workload LXD VM", "name", vmName, "az", availabilityZone, "pool", resourcePool)
	// request allocation
	s.scope.Info("Requesting MAAS allocation", "vm-name", vmName)

	// Use MAAS API to create VM - this is the key communication mechanism
	// MAAS will handle the actual VM creation on the appropriate LXD host
	allocator := s.maasClient.
		Machines().
		Allocator().
		WithCPUCount(*mm.Spec.MinCPU).
		WithMemory(*mm.Spec.MinMemoryInMB)

	// Set zone and resource pool
	if availabilityZone != "" {
		allocator = allocator.WithZone(availabilityZone)
	}
	if resourcePool != "" {
		allocator = allocator.WithResourcePool(resourcePool)
	}

	// Apply placement filters and tags
	if mm.Spec.FailureDomain != nil {
		allocator = allocator.WithZone(*mm.Spec.FailureDomain)
	}
	if mm.Spec.ResourcePool != nil {
		allocator = allocator.WithResourcePool(*mm.Spec.ResourcePool)
	}

	// Add tags only when user supplied
	if len(mm.Spec.Tags) > 0 {
		allocator = allocator.WithTags(mm.Spec.Tags)
	}

	// Allocate the machine (MAAS will create VM on appropriate LXD host)
	m, err := allocator.Allocate(context.TODO())
	if err != nil {
		logVMHostDiagnostics(s, err)
		return nil, errors.Wrapf(err, "failed to allocate workload LXD VM via MAAS API")
	}

	// Set desired hostname before deployment
	if _, err := m.Modifier().SetHostname(vmName).Update(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "failed to set hostname before deploy")
	}

	s.scope.Info("Workload LXD VM allocated", "system-id", m.SystemID(), "host", m.Hostname())
	// Start deployment now
	s.scope.Info("Starting deployment", "system-id", m.SystemID())

	// Configure static IP if specified
	if staticIP != "" {
		if err := s.configureStaticIPForMachine(m, staticIP); err != nil {
			s.scope.Error(err, "failed to configure static IP", "ip", staticIP)
			// Don't fail the entire operation
		}
	}

	// Deploy the VM with user data
	deployingM, err := m.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).Deploy(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deploy workload LXD VM")
	}

	// Set provider ID and system ID
	s.scope.SetProviderID(deployingM.SystemID(), deployingM.Zone().Name())
	if err := s.scope.PatchObject(); err != nil {
		return nil, errors.Wrapf(err, "failed to patch machine with provider ID")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

// getAvailabilityZoneForWorkloadMachine determines the availability zone for a workload machine
func (s *Service) getAvailabilityZoneForWorkloadMachine(poolConfig *infrav1beta1.NodePoolConfig) string {
	// Use pool configuration if available
	if len(poolConfig.AvailabilityZones) > 0 {
		// Simple round-robin assignment across configured AZs
		maasMachines, err := s.scope.ClusterScope.GetClusterMaasMachines()
		if err != nil {
			s.scope.Error(err, "failed to get cluster machines for AZ assignment")
			return poolConfig.AvailabilityZones[0]
		}
		index := len(maasMachines) % len(poolConfig.AvailabilityZones)
		return poolConfig.AvailabilityZones[index]
	}

	// Use machine's failure domain if available
	if s.scope.Machine.Spec.FailureDomain != nil && *s.scope.Machine.Spec.FailureDomain != "" {
		return s.scope.ClusterScope.GetMappedAvailabilityZone(*s.scope.Machine.Spec.FailureDomain)
	}

	// Default to first available AZ
	return "default"
}

// getResourcePoolForWorkloadMachine determines the resource pool for a workload machine
func (s *Service) getResourcePoolForWorkloadMachine(poolConfig *infrav1beta1.NodePoolConfig) string {
	// Use pool configuration if available
	if poolConfig.ResourcePool != nil {
		return s.scope.ClusterScope.GetMappedResourcePool(*poolConfig.ResourcePool)
	}

	// Use machine's resource pool if available
	if s.scope.MaasMachine.Spec.ResourcePool != nil {
		return s.scope.ClusterScope.GetMappedResourcePool(*s.scope.MaasMachine.Spec.ResourcePool)
	}

	// Default resource pool
	return "default"
}

func fromSDKTypeToMachine(m maasclient.Machine) *infrav1beta1.Machine {
	machine := &infrav1beta1.Machine{
		ID:               m.SystemID(),
		Hostname:         m.Hostname(),
		State:            infrav1beta1.MachineState(m.State()),
		Powered:          m.PowerState() == "on",
		AvailabilityZone: m.Zone().Name(),
	}

	if m.FQDN() != "" {
		machine.Addresses = append(machine.Addresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalDNS,
			Address: m.FQDN(),
		})
	}

	for _, v := range m.IPAddresses() {
		machine.Addresses = append(machine.Addresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP,
			Address: v.String(),
		})
	}

	return machine
}

func (s *Service) PowerOnMachine() error {
	_, err := s.maasClient.Machines().Machine(s.scope.GetSystemID()).PowerManagerOn().WithPowerOnComment("maas provider power on").PowerOn(context.Background())
	return err
}

//// ReconcileDNS reconciles the load balancers for the given cluster.
//func (s *Service) ReconcileDNS() error {
//	s.scope.V(2).Info("Reconciling DNS")
//
//	s.scope.SetDNSName("cluster1.maas")
//	return nil
//}
//
