package dns

import (
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
)

// LoadBalancer manages the load balancer for a specific docker cluster.
type Service struct {
	scope *scope.ClusterScope
}

// DNS service returns a new helper for managing a MaaS "DNS" (DNS client loadbalancing)
func NewService(clusterScope *scope.ClusterScope) *Service {
	return &Service{
		scope: clusterScope,
	}
}

// ReconcileLoadbalancers reconciles the load balancers for the given cluster.
func (s *Service) ReconcileLoadbalancers() error {
	s.scope.V(2).Info("Reconciling DNS")

	s.scope.MaasCluster.Status.Network.DNSName = "cluster1.maas"
	return nil
}
