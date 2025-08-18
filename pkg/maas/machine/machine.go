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

	// Decide if we should create a VM via MAAS (LXD) based on user input or node-pool policy.
	// Machine-level enablement (preferred) or node-pool policy (fallback)
	if s.scope.GetDynamicLXD() || s.shouldUseLXDForWorkloadCluster() {
		s.scope.Info("Using LXD VM creation path (unified)", "machine", mm.Name)
		return s.createVMViaMAAS(userDataB64)
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

// createVMViaMAAS performs a unified VM creation flow using the MAAS API.
// It consolidates previous createLXDVM* variants. VM placement is derived from
// MaasMachine spec first, then (if applicable) workload node-pool mappings.
func (s *Service) createVMViaMAAS(userDataB64 string) (*infrav1beta1.Machine, error) {
	ctx := context.TODO()
	mm := s.scope.MaasMachine

	// Determine placement inputs
	var zone string
	if mm.Spec.FailureDomain != nil && *mm.Spec.FailureDomain != "" {
		zone = *mm.Spec.FailureDomain
	} else if s.scope.ClusterScope.IsWorkloadCluster() {
		// Try workload pool-based mapping when FailureDomain not set
		poolConfig := s.scope.ClusterScope.GetNodePoolConfig(s.scope.Machine)
		if poolConfig != nil {
			zone = s.getAvailabilityZoneForWorkloadMachine(poolConfig)
		}
	}

	var resourcePool string
	if mm.Spec.ResourcePool != nil && *mm.Spec.ResourcePool != "" {
		resourcePool = *mm.Spec.ResourcePool
	} else if s.scope.ClusterScope.IsWorkloadCluster() {
		poolConfig := s.scope.ClusterScope.GetNodePoolConfig(s.scope.Machine)
		if poolConfig != nil {
			resourcePool = s.getResourcePoolForWorkloadMachine(poolConfig)
		}
	}

	// Prefer explicit per-machine static IP; fall back to pool-level assignment if present
	staticIP := s.scope.GetStaticIP()
	if staticIP == "" {
		staticIP = s.scope.ClusterScope.GetStaticIPForMachine(s.scope.Machine)
	}

	// Name to set in MAAS for easier tracing
	machineName := s.scope.Machine.Name
	vmName := fmt.Sprintf("vm-%s", machineName)
	if mm.Annotations == nil {
		mm.Annotations = map[string]string{}
	}
	mm.Annotations["maas.spectrocloud.com/vm-name"] = vmName
	_ = s.scope.PatchObject()

	s.scope.Info("Requesting MAAS allocation for VM", "vm-name", vmName, "zone", zone, "pool", resourcePool)

	allocator := s.maasClient.
		Machines().
		Allocator().
		WithCPUCount(*mm.Spec.MinCPU).
		WithMemory(*mm.Spec.MinMemoryInMB)

	if zone != "" {
		allocator = allocator.WithZone(zone)
	}
	if resourcePool != "" {
		allocator = allocator.WithResourcePool(resourcePool)
	}
	if len(mm.Spec.Tags) > 0 {
		allocator = allocator.WithTags(mm.Spec.Tags)
	}

	m, err := allocator.Allocate(ctx)
	if err != nil {
		logVMHostDiagnostics(s, err)
		return nil, errors.Wrapf(err, "failed to allocate VM via MAAS API")
	}

	// Set hostname before deployment
	if _, err := m.Modifier().SetHostname(vmName).Update(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to set hostname before deploy")
	}

	s.scope.Info("Allocated VM", "system-id", m.SystemID())

	// Configure static IP if specified (best-effort; do not fail creation on error)
	if staticIP != "" {
		if err := s.configureStaticIPForMachine(m, staticIP); err != nil {
			s.scope.Error(err, "failed to configure static IP", "ip", staticIP)
		}
	}

	// Deploy the VM with user data
	deployingM, err := m.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).Deploy(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deploy VM")
	}

	// Record providerID and patch
	s.scope.SetProviderID(deployingM.SystemID(), deployingM.Zone().Name())
	if err := s.scope.PatchObject(); err != nil {
		return nil, errors.Wrapf(err, "failed to patch machine with provider ID")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

// createLXDVM creates a new LXD VM and registers it with MAAS
// This method uses MAAS API for cross-cluster communication
// createLXDVM is deprecated; unified creation flow is handled in DeployMachine.
// Keeping a stub for backward compatibility and to minimize churn.
func (s *Service) createLXDVM(userDataB64 string) (*infrav1beta1.Machine, error) {
	return nil, errors.New("createLXDVM is deprecated; use DeployMachine unified flow")
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
// based on user input provided via the MaasMachine annotations (copied from MaasMachineTemplate).
// It does not enforce infra LXD host registration.
func (s *Service) shouldUseLXDForWorkloadCluster() bool {
	if !s.scope.ClusterScope.IsWorkloadCluster() {
		return false
	}
	anns := s.scope.MaasMachine.GetAnnotations()
	if anns == nil {
		return false
	}
	return strings.EqualFold(anns["maas.spectrocloud.com/lxd-enabled"], "true")
}

// createLXDVMForWorkloadCluster creates an LXD VM for a workload cluster machine
// createLXDVMForWorkloadCluster is deprecated; unified creation flow is handled in DeployMachine.
func (s *Service) createLXDVMForWorkloadCluster(userDataB64 string) (*infrav1beta1.Machine, error) {
	return nil, errors.New("createLXDVMForWorkloadCluster is deprecated; use DeployMachine unified flow")
}

// getAvailabilityZoneForWorkloadMachine determines the availability zone for a workload machine
// getAvailabilityZoneForWorkloadMachine is deprecated in favor of shared placement resolver.
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
// getResourcePoolForWorkloadMachine is deprecated in favor of shared placement resolver.
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
