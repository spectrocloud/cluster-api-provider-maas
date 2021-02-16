package machine

import (
	"context"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maasclient"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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

	// TODO verify
	//&infrav1.Machine{
	//	ID:               "hqnsaw",
	//	Hostname:         "enough-bunny",
	//	State:            "Deployed",
	//	Powered:          true,
	//	AvailabilityZone: "az1",
	//	Addresses: []clusterv1.MachineAddress{
	//		{Type: clusterv1.MachineExternalIP, Address: "10.11.130.70"},
	//		{Type: clusterv1.MachineExternalDNS, Address: "enough-bunny.maas"},
	//	},
	//}

	machine := fromSDKTypeToMachine(m)

	return machine, nil
}

func (s *Service) DeployMachine(userDataB64 string) (_ *infrav1.Machine, rerr error) {

	ctx := context.TODO()

	allocateOptions := &maasclient.AllocateMachineOptions{
		// TODO add Resource Pool, CPU, Memory, etc
		AvailabilityZone: pointer.StringPtr("az1"),
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
		DistroSeries: pointer.StringPtr("spectro-u18-k11815"),
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

//// ReconcileLoadbalancers reconciles the load balancers for the given cluster.
//func (s *Service) ReconcileLoadbalancers() error {
//	s.scope.V(2).Info("Reconciling DNS")
//
//	s.scope.SetDNSName("cluster1.maas")
//	return nil
//}
//
