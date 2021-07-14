package dns

import (
	"context"
	"github.com/pkg/errors"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha3"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
	"k8s.io/apimachinery/pkg/util/sets"
)

// LoadBalancer manages the load balancer for a specific docker cluster.
type Service struct {
	scope      *scope.ClusterScope
	maasClient *maasclient.Client
}

var ErrNotFound = errors.New("resource not found")

// DNS service returns a new helper for managing a MaaS "DNS" (DNS client loadbalancing)
func NewService(clusterScope *scope.ClusterScope) *Service {
	return &Service{
		scope:      clusterScope,
		maasClient: scope.NewMaasClient(clusterScope),
	}
}

// ReconcileDNS reconciles the load balancers for the given cluster.
func (s *Service) ReconcileDNS() error {
	s.scope.V(2).Info("Reconciling DNS")

	ctx := context.TODO()

	// get ID of loadbalancer
	dnsResource, err := s.GetDNSResource()
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	dnsName := s.scope.GetDNSName()

	if dnsResource == nil {
		if dnsResource, err = s.maasClient.CreateDNSResources(ctx, maasclient.CreateDNSResourcesOptions{
			FQDN:        s.scope.GetDNSName(),
			AddressTTL:  "10",
			IpAddresses: nil,
		}); err != nil {
			return errors.Wrapf(err, "Unable to create DNS Resources")
		}
	}

	s.scope.SetDNSName(dnsName)

	return nil
}

// UpdateAttachments reconciles the load balancers for the given cluster.
func (s *Service) UpdateDNSAttachments(IPs []string) error {
	s.scope.V(2).Info("Reconciling DNS")

	ctx := context.TODO()

	// get ID of loadbalancer
	dnsResource, err := s.GetDNSResource()
	if err != nil {
		return err
	}

	updateOptions := maasclient.UpdateDNSResourcesOptions{
		ID:          dnsResource.ID,
		IpAddresses: IPs,
	}

	if _, err = s.maasClient.UpdateDNSResources(ctx, updateOptions); err != nil {
		return errors.Wrap(err, "Unable to update IPs")
	}

	return nil
}

// TODO do at some point
//func MachineIsRunning(m *infrav1.MaasMachine) bool {
//	if !m.Status.MachinePowered {
//		return false
//	}
//
//	//allMachinePodConditions := []clusterv1.ConditionType{
//	//	controlplanev1.MachineAPIServerPodHealthyCondition,
//	//	controlplanev1.MachineControllerManagerPodHealthyCondition,
//	//	controlplanev1.MachineSchedulerPodHealthyCondition,
//	//}
//	//if controlPlane.IsEtcdManaged() {
//	//	allMachinePodConditions = append(allMachinePodConditions, controlplanev1.MachineEtcdPodHealthyCondition)
//	//}
//
//}

// InstanceIsRegisteredWithAPIServerELB returns true if the instance is already registered with the APIServer ELB.
func (s *Service) MachineIsRegisteredWithAPIServerDNS(i *infrav1.Machine) (bool, error) {
	ips, err := s.GetAPIServerDNSRecords()
	if err != nil {
		return false, err
	}

	for _, mAddress := range i.Addresses {
		if ips.Has(mAddress.Address) {
			return true, nil
		}
	}

	return false, nil
}

func (s *Service) GetAPIServerDNSRecords() (sets.String, error) {
	dnsResource, err := s.GetDNSResource()
	if err != nil {
		return nil, err
	}

	ips := sets.NewString()
	for _, address := range dnsResource.IpAddresses {
		if address.IpAddress != "" {
			ips.Insert(address.IpAddress)
		}
	}

	return ips, nil
}

func (s *Service) GetDNSResource() (*maasclient.DNSResource, error) {
	dnsName := s.scope.GetDNSName()
	if dnsName == "" {
		return nil, errors.New("No DNS on the cluster set!")
	}

	options := &maasclient.GetDNSResourcesOptions{
		FQDN: &dnsName,
	}

	d, err := s.maasClient.GetDNSResources(context.TODO(), options)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving dns resources %q", dnsName)
	} else if len(d) > 1 {
		return nil, errors.Errorf("expected 1 DNS Resource for %q, got %d", dnsName, len(d))
	} else if len(d) == 0 {
		return nil, ErrNotFound
	}

	return d[0], nil
}
