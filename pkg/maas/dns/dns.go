package dns

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	infrainfrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Service struct {
	scope      *scope.ClusterScope
	maasClient maasclient.ClientSetInterface
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

	if s.scope.IsCustomEndpoint() {
		return nil
	}

	ctx := context.TODO()

	dnsResource, err := s.GetDNSResource()
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	dnsName := s.scope.GetDNSName()

	if dnsResource == nil {
		if _, err = s.maasClient.DNSResources().
			Builder().
			WithFQDN(s.scope.GetDNSName()).
			WithAddressTTL("10").
			WithIPAddresses(nil).
			Create(ctx); err != nil {
			return errors.Wrapf(err, "Unable to create DNS Resources")
		}
	}

	s.scope.SetDNSName(dnsName)

	return nil
}

// UpdateDNSAttachmentsWithResource updates DNS attachments using a pre-fetched DNS resource,
// avoiding additional GET API calls. Returns true if an update was performed.
func (s *Service) UpdateDNSAttachmentsWithResource(dnsResource maasclient.DNSResource, IPs []string) (bool, error) {
	s.scope.V(2).Info("Updating DNS Attachments with pre-fetched resource")

	if s.scope.IsCustomEndpoint() {
		return false, nil
	}

	return s.updateResourceIPs(dnsResource, IPs)
}

// updateResourceIPs applies desired IPs to a given DNS resource with idempotency; returns true if an update was made.
func (s *Service) updateResourceIPs(dnsResource maasclient.DNSResource, IPs []string) (bool, error) {
	ctx := context.TODO()

	// Build desired set (dedupe, ignore empties)
	desired := sets.NewString()
	for _, ip := range IPs {
		if ip != "" {
			desired.Insert(ip)
		}
	}

	// Build current set from resource
	current := sets.NewString()
	for _, addr := range dnsResource.IPAddresses() {
		if a := addr.IP().String(); a != "" {
			current.Insert(a)
		}
	}

	if desired.Equal(current) {
		s.scope.V(4).Info("DNS attachments up-to-date; skipping MAAS update")
		return false, nil
	}

	desiredList := desired.UnsortedList()
	sort.Strings(desiredList)

	if _, err := dnsResource.Modifier().SetIPAddresses(desiredList).Modify(ctx); err != nil {
		return false, errors.Wrap(err, "Unable to update IPs")
	}

	s.scope.V(2).Info("DNS attachments updated successfully")
	return true, nil
}

// TODO do at some point
//func MachineIsRunning(m *infrainfrav1beta1.MaasMachine) bool {
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
func (s *Service) MachineIsRegisteredWithAPIServerDNS(i *infrainfrav1beta1.Machine) (bool, error) {
	if s.scope.IsCustomEndpoint() {
		return true, nil
	}

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

	if s.scope.IsCustomEndpoint() {
		return nil, nil
	}

	dnsResource, err := s.GetDNSResource()
	if err != nil {
		return nil, err
	}

	ips := sets.NewString()
	for _, address := range dnsResource.IPAddresses() {
		if address.IP().String() != "" {
			ips.Insert(address.IP().String())
		}
	}

	return ips, nil
}

func (s *Service) GetDNSResource() (maasclient.DNSResource, error) {

	if s.scope.IsCustomEndpoint() {
		return nil, nil
	}

	dnsName := s.scope.GetDNSName()
	if dnsName == "" {
		return nil, errors.New("No DNS on the cluster set!")
	}

	d, err := s.maasClient.DNSResources().
		List(context.Background(),
			maasclient.ParamsBuilder().Set(maasclient.FQDNKey, dnsName))
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving dns resources %q", dnsName)
	} else if len(d) > 1 {
		return nil, errors.Errorf("expected 1 DNS Resource for %q, got %d", dnsName, len(d))
	} else if len(d) == 0 {
		return nil, ErrNotFound
	}

	return d[0], nil
}
