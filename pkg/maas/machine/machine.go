package machine

import (
	"context"
	"fmt"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maasclient"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

// Service manages the MaaS machine
type Service struct {
	scope *scope.MachineScope
	maasClient *maasclient.Client
}

// DNS service returns a new helper for managing a MaaS "DNS" (DNS client loadbalancing)
func NewService(machineScope *scope.MachineScope) *Service {
	return &Service{
		scope: machineScope,
		maasClient: scope.NewMaasClient(machineScope.ClusterScope),
	}
}

func (s *Service) GetMachine(systemID string) (*infrav1.Machine, error) {

	dnsResources, err := s.maasClient.GetDNSResources(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("hello", dnsResources)

	machine := &infrav1.Machine{
		ID:               "hqnsaw",
		Hostname:         "enough-bunny",
		State:            "Deployed",
		Powered:          true,
		AvailabilityZone: "az1",
		Addresses: []clusterv1.MachineAddress{
			{Type: clusterv1.MachineExternalIP, Address: "10.11.130.70"},
			{Type: clusterv1.MachineExternalDNS, Address: "enough-bunny.maas"},
		},
	}

	return machine, nil
}

//// ReconcileLoadbalancers reconciles the load balancers for the given cluster.
//func (s *Service) ReconcileLoadbalancers() error {
//	s.scope.V(2).Info("Reconciling DNS")
//
//	s.scope.SetDNSName("cluster1.maas")
//	return nil
//}
//
