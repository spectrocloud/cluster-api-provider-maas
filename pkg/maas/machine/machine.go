package machine

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
	"k8s.io/klog/v2/textlogger"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Service manages the MaaS machine
type Service struct {
	scope      *scope.MachineScope
	maasClient maasclient.ClientSetInterface
	lxdService *lxd.Service
}

// NewService returns a new helper for managing a MaaS machine
func NewService(machineScope *scope.MachineScope) *Service {
	return &Service{
		scope:      machineScope,
		maasClient: scope.NewMaasClient(machineScope.ClusterScope),
		lxdService: lxd.NewService(machineScope),
	}
}

func (s *Service) GetMachine(systemID string) (*infrav1beta1.Machine, error) {
	if systemID == "" {
		return nil, nil
	}

	// Check if this is an LXD machine
	if s.scope.IsLXDProviderID() {
		return s.getLXDMachineFromProviderID()
	}

	// Original bare metal logic
	m, err := s.maasClient.Machines().Machine(systemID).Get(context.Background())
	if err != nil {
		return nil, err
	}

	machine := fromSDKTypeToMachine(m)

	return machine, nil
}

func (s *Service) ReleaseMachine(systemID string) error {
	// Check if this is an LXD machine
	if s.scope.IsLXDProviderID() {
		return s.releaseLXDMachine()
	}

	// Original bare metal release logic
	return s.releaseBaremetalMachine(systemID)
}

// releaseBaremetalMachine releases a bare metal machine
func (s *Service) releaseBaremetalMachine(systemID string) error {
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

// releaseLXDMachine releases an LXD VM
func (s *Service) releaseLXDMachine() error {
	ctx := context.TODO()

	vmName := s.scope.GetLXDVMNameFromProviderID()
	hostSystemID := s.scope.GetLXDHostSystemID()

	if vmName == "" || hostSystemID == "" {
		return errors.New("invalid LXD provider ID format for release")
	}

	s.scope.Info("Releasing LXD VM", "vm-name", vmName, "host-id", hostSystemID)

	// For LXD VMs, use the VM system ID (vmName) to delete
	err := s.lxdService.DeleteVM(ctx, vmName)
	if err != nil {
		// Check if VM is already gone (implement error checking later)
		s.scope.Info("LXD VM deletion completed", "vm-name", vmName)
		return errors.Wrap(err, "failed to delete LXD VM")
	}

	s.scope.Info("LXD VM released successfully", "vm-name", vmName)
	return nil
}

func (s *Service) DeployMachine(userDataB64 string) (_ *infrav1beta1.Machine, rerr error) {
	// Check if this is LXD provisioning
	if s.scope.IsLXDProvisioning() {
		return s.deployLXDMachine(userDataB64)
	}

	// Original bare metal deployment logic
	return s.deployBaremetalMachine(userDataB64)
}

// deployBaremetalMachine deploys a bare metal machine
func (s *Service) deployBaremetalMachine(userDataB64 string) (_ *infrav1beta1.Machine, rerr error) {
	ctx := context.TODO()
	log := textlogger.NewLogger(textlogger.NewConfig())

	mm := s.scope.MaasMachine

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

		m, err = allocator.Allocate(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to allocate machine")
		}

		s.scope.SetProviderID(m.SystemID(), m.Zone().Name())
		err = s.scope.PatchObject()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to patch machine with provider id")
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

	deployingM, err := m.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).Deploy(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to deploy machine")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

// deployLXDMachine deploys an LXD VM on a selected host
func (s *Service) deployLXDMachine(userDataB64 string) (_ *infrav1beta1.Machine, rerr error) {
	ctx := context.TODO()
	s.scope.Info("Starting LXD machine deployment")

	mm := s.scope.MaasMachine
	lxdConfig := s.scope.GetLXDConfig()
	if lxdConfig == nil {
		return nil, errors.New("LXD configuration is required for LXD provisioning")
	}

	// Check if we already have a provider ID (re-deployment scenario)
	if s.scope.GetProviderID() != "" {
		return s.getLXDMachineFromProviderID()
	}

	// Get available LXD hosts
	hosts, err := s.lxdService.GetAvailableLXDHosts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available LXD hosts")
	}

	if len(hosts) == 0 {
		return nil, errors.New("no LXD hosts available")
	}

	// Select optimal host
	host, err := s.lxdService.SelectOptimalHost(ctx, hosts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select LXD host")
	}

	s.scope.Info("Selected LXD host", "host-id", host.SystemID, "host-name", host.Hostname)

	// Create VM specification
	vmSpec := &lxd.VMSpec{
		Cores:    2,    // Default cores
		Memory:   4096, // Default 4GB memory
		UserData: userDataB64,
		HostID:   host.SystemID,
		Profile:  "default",
		Project:  "default",
		Tags:     mm.Spec.Tags,
	}

	// Apply LXD resource configuration if provided
	if lxdConfig.ResourceAllocation != nil {
		if lxdConfig.ResourceAllocation.CPU != nil {
			vmSpec.Cores = *lxdConfig.ResourceAllocation.CPU
		}
		if lxdConfig.ResourceAllocation.Memory != nil {
			vmSpec.Memory = *lxdConfig.ResourceAllocation.Memory
		}
		if lxdConfig.ResourceAllocation.Disk != nil {
			// Add disk specification
			vmSpec.Disks = []lxd.DiskSpec{
				{Size: fmt.Sprintf("%dGB", *lxdConfig.ResourceAllocation.Disk), Pool: "default"},
			}
		}
	}

	// Compose the LXD VM
	lxdResult, err := s.lxdService.ComposeVM(ctx, vmSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compose LXD VM")
	}

	s.scope.Info("Composed LXD VM", "system-id", lxdResult.SystemID, "host-id", lxdResult.HostID)

	// Deploy the VM with user data
	vm, err := s.lxdService.DeployVM(ctx, lxdResult.SystemID, userDataB64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy LXD VM")
	}

	// Set provider ID for LXD VM
	s.scope.SetLXDProviderID(lxdResult.SystemID, lxdResult.HostID, lxdResult.FailureDomain)

	// Update LXD status
	s.scope.SetLXDStatus(lxdResult.HostID, lxdResult.SystemID, "", lxdConfig.ResourceAllocation)

	// Patch the machine object
	err = s.scope.PatchObject()
	if err != nil {
		return nil, errors.Wrap(err, "unable to patch machine with LXD provider ID")
	}

	s.scope.Info("LXD machine deployment completed", "system-id", lxdResult.SystemID)
	return vm, nil
}

// getLXDMachineFromProviderID retrieves LXD machine info from existing provider ID
func (s *Service) getLXDMachineFromProviderID() (*infrav1beta1.Machine, error) {
	ctx := context.TODO()

	vmName := s.scope.GetLXDVMNameFromProviderID()
	hostSystemID := s.scope.GetLXDHostSystemID()

	if vmName == "" || hostSystemID == "" {
		return nil, errors.New("invalid LXD provider ID format")
	}

	// For LXD VMs, use the VM system ID (vmName) to get status
	vm, err := s.lxdService.GetVM(ctx, vmName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get LXD VM status")
	}

	return vm, nil
}

// getFailureDomain returns the failure domain for the machine
func (s *Service) getFailureDomain() *string {
	mm := s.scope.MaasMachine
	failureDomain := mm.Spec.FailureDomain
	if failureDomain == nil {
		if s.scope.Machine.Spec.FailureDomain != nil && *s.scope.Machine.Spec.FailureDomain != "" {
			failureDomain = s.scope.Machine.Spec.FailureDomain
		}
	}
	return failureDomain
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
	// Check if this is an LXD machine
	if s.scope.IsLXDProviderID() {
		return s.powerOnLXDMachine()
	}

	// Original bare metal logic
	_, err := s.maasClient.Machines().Machine(s.scope.GetSystemID()).PowerManagerOn().WithPowerOnComment("maas provider power on").PowerOn(context.Background())
	return err
}

// powerOnLXDMachine powers on an LXD VM
// For LXD VMs created through MAAS composition, power control is handled through MAAS
func (s *Service) powerOnLXDMachine() error {
	ctx := context.Background()

	vmName := s.scope.GetLXDVMNameFromProviderID()
	hostSystemID := s.scope.GetLXDHostSystemID()

	if vmName == "" || hostSystemID == "" {
		return errors.New("invalid LXD provider ID format for power on")
	}

	s.scope.Info("Powering on LXD VM through MAAS", "vm-name", vmName, "host-id", hostSystemID)

	// For composed LXD VMs, use MAAS power management with the VM system ID
	_, err := s.maasClient.Machines().Machine(vmName).PowerManagerOn().WithPowerOnComment("maas-lxd provider power on").PowerOn(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to power on LXD VM")
	}

	s.scope.Info("LXD VM powered on successfully", "vm-name", vmName)
	return nil
}

//// ReconcileDNS reconciles the load balancers for the given cluster.
//func (s *Service) ReconcileDNS() error {
//	s.scope.V(2).Info("Reconciling DNS")
//
//	s.scope.SetDNSName("cluster1.maas")
//	return nil
//}
//
