package machine

import (
	"context"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha3"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maasclient"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// Service manages the MaaS machine
type Service struct {
	scope      *scope.MachineScope
	maasClient *maasclient.Client
}

// DNS service returns a new helper for managing a MaaS "DNS" (DNS client loadbalancing)
func NewService(machineScope *scope.MachineScope) *Service {
	return &Service{
		scope:      machineScope,
		maasClient: scope.NewMaasClient(machineScope.ClusterScope),
	}
}

func (s *Service) GetMachine(systemID string) (*infrav1.Machine, error) {

	m, err := s.maasClient.GetMachine(context.TODO(), systemID)
	if err != nil {
		return nil, err
	}

	machine := fromSDKTypeToMachine(m)

	return machine, nil
}

func (s *Service) ReleaseMachine(systemID string) error {
	ctx := context.TODO()

	err := s.maasClient.ReleaseMachine(ctx, systemID)
	if err != nil {
		return errors.Wrapf(err, "Unable to release machine")
	}

	return nil
}

func (s *Service) DeployMachine(userDataB64 string) (_ *infrav1.Machine, rerr error) {

	ctx := context.TODO()

	mm := s.scope.MaasMachine
	failureDomain := s.scope.Machine.Spec.FailureDomain

	allocateOptions := &maasclient.AllocateMachineOptions{
		// TODO add Resource Pool, CPU, Memory, etc
		AvailabilityZone: failureDomain,

		ResourcePool: mm.Spec.ResourcePool,
		MinCPU:       mm.Spec.MinCPU,
		MinMem:       mm.Spec.MinMemory,
	}

	m, err := s.maasClient.AllocateMachine(ctx, allocateOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to allocate machine")
	}

	s.scope.Info("Allocated machine", "system-id", m.SystemID)

	defer func() {
		if rerr != nil {
			s.scope.Info("Attempting to release machine which failed to deploy")
			err := s.maasClient.ReleaseMachine(ctx, m.SystemID)
			if err != nil {
				// Is it right to NOT set rerr so we can see the original issue?
				log.Error(err, "Unable to release properly")
			}
		}
	}()

	noSwap := 0
	updateOptions := maasclient.UpdateMachineOptions{
		SystemID: m.SystemID,
		SwapSize: &noSwap,
	}
	if _, err := s.maasClient.UpdateMachine(ctx, updateOptions); err != nil {
		return nil, errors.Wrapf(err, "Unable to disable swap")
	}

	s.scope.Info("Swap disabled", "system-id", m.SystemID)

	deployOptions := maasclient.DeployMachineOptions{
		SystemID:     m.SystemID,
		UserData:     pointer.StringPtr(userDataB64),
		OSSystem:     pointer.StringPtr("custom"),
		DistroSeries: &mm.Spec.Image,
	}

	deployingM, err := s.maasClient.DeployMachine(ctx, deployOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to deploy machine")
	}

	return fromSDKTypeToMachine(deployingM), nil
}

func fromSDKTypeToMachine(m *maasclient.Machine) *infrav1.Machine {
	machine := &infrav1.Machine{
		ID:               m.SystemID,
		Hostname:         m.Hostname,
		State:            infrav1.MachineState(m.State),
		Powered:          m.PowerState == "on",
		AvailabilityZone: m.AvailabilityZone,
	}

	if m.FQDN != "" {
		machine.Addresses = append(machine.Addresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalDNS,
			Address: m.FQDN,
		})
	}

	for _, v := range m.IpAddresses {
		machine.Addresses = append(machine.Addresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP,
			Address: v,
		})
	}

	return machine
}

//// ReconcileDNS reconciles the load balancers for the given cluster.
//func (s *Service) ReconcileDNS() error {
//	s.scope.V(2).Info("Reconciling DNS")
//
//	s.scope.SetDNSName("cluster1.maas")
//	return nil
//}
//
