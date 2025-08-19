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

	// Configure static IP before deployment
	if staticIP := s.scope.GetStaticIP(); staticIP != "" {
		staticIPConfig := s.scope.GetStaticIPConfig()
		if staticIPConfig != nil {
			err := s.setMachineStaticIP(m.SystemID(), staticIPConfig)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to configure static IP")
			}
		}
	}

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

// setMachineStaticIP configures static IP for a machine using the simplified networkInterfaceImpl branch API
func (s *Service) setMachineStaticIP(systemID string, config *infrav1beta1.StaticIPConfig) error {
	ctx := context.TODO()

	// Use the new simplified API to set static IP on boot interface
	err := s.maasClient.NetworkInterfaces().SetBootInterfaceStaticIP(ctx, systemID, config.IP)
	if err != nil {
		return fmt.Errorf("failed to set static IP %s on boot interface for machine %s: %w", config.IP, systemID, err)
	}

	s.scope.Info("Static IP configured", "ip", config.IP, "systemID", systemID)
	return nil
}

// createBootInterfaceBridge creates a bridge on the boot interface using maas-client-go
// First checks if the boot interface type is "physical" before attempting to create a bridge
func (s *Service) createBootInterfaceBridge(ctx context.Context, systemID string) error {
	s.scope.Info("Checking boot interface type", "systemID", systemID)

	// First, check if the boot interface is physical using GetBootInterfaceType
	machine, err := s.maasClient.Machines().Machine(systemID).Get(ctx)
	if err != nil {
		s.scope.Error(err, "Failed to get machine details")
	}
	interfaceType := machine.GetBootInterfaceType()
	s.scope.Info("Boot interface type", "systemID", systemID, "interfaceType", interfaceType)

	// Only create bridge if the boot interface is physical
	if interfaceType != "physical" {
		s.scope.Info("Boot interface is not physical, skipping bridge creation",
			"systemID", systemID, "interfaceType", interfaceType)
		return nil
	}

	s.scope.Info("Creating bridge for physical boot interface", "systemID", systemID, "interfaceType", interfaceType)

	// Now create the bridge since we know it's physical
	_, err = s.maasClient.NetworkInterfaces().CreateBootInterfaceBridge(ctx, systemID, "br0")
	if err != nil {
		// Handle expected errors gracefully (e.g., bridge already exists)
		if strings.Contains(err.Error(), "already bridged") ||
			strings.Contains(err.Error(), "already exists") {
			s.scope.V(1).Info("Boot interface bridge creation skipped", "systemID", systemID, "reason", err.Error())
			return nil
		}
		return fmt.Errorf("failed to create boot interface bridge for machine %s: %w", systemID, err)
	}

	s.scope.Info("Boot interface bridge created successfully", "systemID", systemID, "bridgeName", "br0")
	return nil
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
