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
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

// Service manages the MaaS machine
var (
	ErrBrokenMachine = errors.New("broken machine encountered")
	ErrVMComposing   = errors.New("vm composing/commissioning")
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

	// Decide if we should create a VM via MAAS (LXD) based on machine-level enablement only.
	if s.scope.GetDynamicLXD() {
		s.scope.Info("Using LXD VM creation path (unified)", "machine", mm.Name)
		return s.createVMViaMAAS(ctx, userDataB64)
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
func (s *Service) createVMViaMAAS(ctx context.Context, userDataB64 string) (*infrav1beta1.Machine, error) {

	mm := s.scope.MaasMachine

	// If a VM was already composed earlier (providerID/system-id present), reuse it and only deploy
	if id := s.scope.GetInstanceID(); id != nil && *id != "" {
		m, err := s.maasClient.Machines().Machine(*id).Get(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get existing VM by system-id")
		}
		// Best-effort: set hostname and static IP before deploy
		machineName := s.scope.Machine.Name
		vmName := fmt.Sprintf("vm-%s", machineName)
		_, _ = m.Modifier().SetHostname(vmName).Update(ctx)
		if staticIP := s.scope.GetStaticIP(); staticIP != "" {
			_ = s.configureStaticIPForMachine(m, staticIP)
		}
		deployingM, err := m.Deployer().
			SetUserData(userDataB64).
			SetOSSystem("custom").
			SetDistroSeries(mm.Spec.Image).Deploy(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to deploy existing VM")
		}
		// Determine fallback zone
		fallbackZone := ""
		if deployingM.Zone() != nil {
			fallbackZone = deployingM.Zone().Name()
		}
		if fallbackZone == "" {
			if mm.Spec.FailureDomain != nil && *mm.Spec.FailureDomain != "" {
				fallbackZone = *mm.Spec.FailureDomain
			} else if s.scope.Machine.Spec.FailureDomain != nil && *s.scope.Machine.Spec.FailureDomain != "" {
				fallbackZone = *s.scope.Machine.Spec.FailureDomain
			}
		}
		s.scope.SetSystemID(deployingM.SystemID())
		s.scope.SetProviderID(deployingM.SystemID(), fallbackZone)
		if fallbackZone != "" {
			s.scope.SetFailureDomain(fallbackZone)
		}
		_ = s.scope.PatchObject()
		res := fromSDKTypeToMachine(deployingM)
		if res.AvailabilityZone == "" {
			res.AvailabilityZone = fallbackZone
		}
		return res, nil
	}

	// No composed VM yet; wait for PrepareLXDVM/commissioning to complete
	if _, err := s.PrepareLXDVM(); err != nil {
		return nil, errors.Wrap(err, "compose failed prior to deploy")
	}
	conditions.MarkFalse(s.scope.MaasMachine, infrav1beta1.MachineDeployedCondition, infrav1beta1.MachineDeployingReason, clusterv1.ConditionSeverityInfo, "VM composed; commissioning")
	_ = s.scope.PatchObject()
	return nil, ErrVMComposing
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

// createLXDVMForWorkloadCluster creates an LXD VM for a workload cluster machine
// createLXDVMForWorkloadCluster is deprecated; unified creation flow is handled in DeployMachine.
func (s *Service) createLXDVMForWorkloadCluster(userDataB64 string) (*infrav1beta1.Machine, error) {
	return nil, errors.New("createLXDVMForWorkloadCluster is deprecated; use DeployMachine unified flow")
}

// PrepareLXDVM composes an LXD VM and sets providerID; it does not deploy/boot the VM.
func (s *Service) PrepareLXDVM() (*infrav1beta1.Machine, error) {
	ctx := context.TODO()
	mm := s.scope.MaasMachine

	// If already composed (system-id or providerID present), reuse
	if mm.Spec.SystemID != nil && *mm.Spec.SystemID != "" {
		m, err := s.maasClient.Machines().Machine(*mm.Spec.SystemID).Get(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get existing VM by system-id (pre-bootstrap)")
		}
		return fromSDKTypeToMachine(m), nil
	}

	if id := s.scope.GetInstanceID(); id != nil && *id != "" {
		m, err := s.maasClient.Machines().Machine(*id).Get(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get existing VM by system-id (pre-bootstrap)")
		}
		return fromSDKTypeToMachine(m), nil
	}

	// Determine placement inputs using only machine-level fields
	var zone string
	if mm.Spec.FailureDomain != nil && *mm.Spec.FailureDomain != "" {
		zone = *mm.Spec.FailureDomain
	} else if s.scope.Machine.Spec.FailureDomain != nil && *s.scope.Machine.Spec.FailureDomain != "" {
		zone = *s.scope.Machine.Spec.FailureDomain
	}

	var resourcePool string
	if mm.Spec.ResourcePool != nil && *mm.Spec.ResourcePool != "" {
		resourcePool = *mm.Spec.ResourcePool
	}

	// VM name and minimal resources
	vmName := mm.Annotations["maas.spectrocloud.com/vm-name"]
	if vmName == "" {
		uid := string(s.scope.Machine.UID)
		short := uid
		if len(uid) >= 5 {
			short = uid[:5]
		}
		vmName = fmt.Sprintf("vm-%s-%s", s.scope.Machine.Name, short)
		if mm.Annotations == nil {
			mm.Annotations = map[string]string{}
		}
		mm.Annotations["maas.spectrocloud.com/vm-name"] = vmName
		_ = s.scope.PatchObject()
	}
	cpu := 1
	mem := 1024
	if mm.Spec.MinCPU != nil && *mm.Spec.MinCPU > 0 {
		cpu = *mm.Spec.MinCPU
	}
	if mm.Spec.MinMemoryInMB != nil && *mm.Spec.MinMemoryInMB > 0 {
		mem = *mm.Spec.MinMemoryInMB
	}
	// // Clamp memory to MAAS 10GiB per-VM compose cap
	// if mem > 10240 {
	// 	mem = 10240
	// }

	// Enforce minimum 60GB storage
	diskSizeGB := 60
	if mm.Spec.LXD != nil && mm.Spec.LXD.VMConfig != nil && mm.Spec.LXD.VMConfig.DiskSize != nil && *mm.Spec.LXD.VMConfig.DiskSize > diskSizeGB {
		diskSizeGB = *mm.Spec.LXD.VMConfig.DiskSize
	}

	params := maasclient.ParamsBuilder().
		Set("hostname", vmName).
		Set("cores", fmt.Sprintf("%d", cpu)).
		Set("memory", fmt.Sprintf("%d", mem)).
		Set("storage", fmt.Sprintf("%d", diskSizeGB))

	// Select an LXD VM host based on zone and resource pool
	hosts, err := s.maasClient.VMHosts().List(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list LXD VM hosts")
	}
	selectedHost, err := lxd.SelectLXDHostWithMaasClient(hosts, zone, resourcePool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select LXD VM host")
	}

	// Create the VM on the selected host
	m, err := selectedHost.Composer().Compose(ctx, params)
	if err != nil {
		// If hostname already exists, reuse that VM
		errStr := err.Error()
		if strings.Contains(strings.ToLower(errStr), "hostname") && strings.Contains(strings.ToLower(errStr), "already exists") {
			// First try global machines list
			if all, aerr := s.maasClient.Machines().List(ctx, nil); aerr == nil {
				for _, cand := range all {
					cid := cand.SystemID()
					if cid == "" {
						continue
					}
					cDet, cg := s.maasClient.Machines().Machine(cid).Get(ctx)
					if cg == nil && strings.EqualFold(cDet.Hostname(), vmName) {
						s.scope.SetSystemID(cDet.SystemID())
						s.scope.SetProviderID(cDet.SystemID(), zone)
						if zone != "" {
							s.scope.SetFailureDomain(zone)
						}
						_ = s.scope.PatchObject()
						s.scope.Info("Reusing existing VM by hostname (pre-bootstrap)", "system-id", cDet.SystemID())
						res := fromSDKTypeToMachine(cDet)
						if res.AvailabilityZone == "" {
							res.AvailabilityZone = zone
						}
						return res, nil
					}
				}
			}
			// Then try host-local list
			if list, lerr := selectedHost.Machines().List(ctx); lerr == nil {
				for _, ex := range list {
					exID := ex.SystemID()
					if exID == "" {
						continue
					}
					// fetch details to get hostname
					exDet, gerr := s.maasClient.Machines().Machine(exID).Get(ctx)
					if gerr != nil {
						continue
					}
					if strings.EqualFold(exDet.Hostname(), vmName) {
						s.scope.SetSystemID(exDet.SystemID())
						s.scope.SetProviderID(exDet.SystemID(), zone)
						if zone != "" {
							s.scope.SetFailureDomain(zone)
						}
						_ = s.scope.PatchObject()
						s.scope.Info("Reusing existing VM by hostname (pre-bootstrap)", "system-id", exDet.SystemID())
						res := fromSDKTypeToMachine(exDet)
						if res.AvailabilityZone == "" {
							res.AvailabilityZone = zone
						}
						return res, nil
					}
				}
			}
			return nil, errors.Wrap(err, "failed to compose VM on LXD host")
		}
	}

	// Set IDs early so system-id/providerID are recorded
	if m.SystemID() != "" {
		s.scope.SetSystemID(m.SystemID())
		s.scope.SetProviderID(m.SystemID(), zone)
		if zone != "" {
			s.scope.SetFailureDomain(zone)
		}
		_ = s.scope.PatchObject()
	}
	s.scope.Info("Composed VM (pre-bootstrap)", "system-id", m.SystemID())

	res := fromSDKTypeToMachine(m)
	if res.AvailabilityZone == "" {
		res.AvailabilityZone = zone
	}
	return res, nil
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
