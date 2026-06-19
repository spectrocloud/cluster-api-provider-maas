package dns

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	mockclientset "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/client/mock"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// fakeClientSet minimally satisfies maasclient.ClientSetInterface for tests
type fakeClientSet struct{ dns maasclient.DNSResources }

func (f *fakeClientSet) BootResources() maasclient.BootResources         { return nil }
func (f *fakeClientSet) DNSResources() maasclient.DNSResources           { return f.dns }
func (f *fakeClientSet) Domains() maasclient.Domains                     { return nil }
func (f *fakeClientSet) IPAddresses() maasclient.IPAddresses             { return nil }
func (f *fakeClientSet) Tags() maasclient.Tags                           { return nil }
func (f *fakeClientSet) Machines() maasclient.Machines                   { return nil }
func (f *fakeClientSet) NetworkInterfaces() maasclient.NetworkInterfaces { return nil }
func (f *fakeClientSet) RackControllers() maasclient.RackControllers     { return nil }
func (f *fakeClientSet) ResourcePools() maasclient.ResourcePools         { return nil }
func (f *fakeClientSet) Spaces() maasclient.Spaces                       { return nil }
func (f *fakeClientSet) Users() maasclient.Users                         { return nil }
func (f *fakeClientSet) Zones() maasclient.Zones                         { return nil }
func (f *fakeClientSet) SSHKeys() maasclient.SSHKeys                     { return nil }
func (f *fakeClientSet) VMHosts() maasclient.VMHosts                     { return nil }
func (f *fakeClientSet) IPRanges() maasclient.IPRanges                   { return nil }
func (f *fakeClientSet) Subnets() maasclient.Subnets                     { return nil }

// fakeIPAddress satisfies maasclient.IPAddress for tests
type fakeIPAddress struct{ ip net.IP }

func (f *fakeIPAddress) IP() net.IP                                  { return f.ip }
func (f *fakeIPAddress) InterfaceSet() []maasclient.NetworkInterface { return nil }

func TestDNS(t *testing.T) {
	log := klogr.New()
	cluster := &v1beta1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "a",
		},
	}
	maasCluster := &infrav1beta1.MaasCluster{
		Spec: infrav1beta1.MaasClusterSpec{
			DNSDomain: "b.com",
		},
	}

	t.Run("reconcile dns", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
		mockDNSResources := mockclientset.NewMockDNSResources(ctrl)
		mockDNSResourceBuilder := mockclientset.NewMockDNSResourceBuilder(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: mockClientSetInterface,
		}
		mockClientSetInterface.EXPECT().DNSResources().Return(mockDNSResources)
		mockDNSResources.EXPECT().List(context.Background(), gomock.Any()).Return(nil, nil)
		mockClientSetInterface.EXPECT().DNSResources().Return(mockDNSResources)
		mockDNSResources.EXPECT().Builder().Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithFQDN(gomock.Any()).Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithAddressTTL("10").Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithIPAddresses(nil).Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().Create(context.TODO())
		err := s.ReconcileDNS()

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.scope.GetDNSName()).To(ContainSubstring(cluster.Name))
		g.Expect(s.scope.GetDNSName()).To(ContainSubstring(maasCluster.Spec.DNSDomain))
	})

	t.Run("update dns attachment", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
		mockDNSResources := mockclientset.NewMockDNSResources(ctrl)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		mockDNSResourceModifier := mockclientset.NewMockDNSResourceModifier(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: mockClientSetInterface,
		}

		mockClientSetInterface.EXPECT().DNSResources().Return(mockDNSResources)
		mockDNSResources.EXPECT().List(context.Background(), gomock.Any()).Return([]maasclient.DNSResource{mockDNSResource}, nil)
		mockDNSResource.EXPECT().Modifier().Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().SetIPAddresses([]string{"1.1.1.1", "8.8.8.8"}).Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().Modify(context.TODO()).Return(mockDNSResource, nil)

		err := s.UpdateDNSAttachments([]string{"1.1.1.1", "8.8.8.8"})

		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("machine is registered", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
		mockDNSResources := mockclientset.NewMockDNSResources(ctrl)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		mockIPAddress := mockclientset.NewMockIPAddress(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: mockClientSetInterface,
		}
		mockClientSetInterface.EXPECT().DNSResources().Return(mockDNSResources)
		mockDNSResources.EXPECT().List(context.Background(), gomock.Any()).Return([]maasclient.DNSResource{mockDNSResource}, nil)
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{mockIPAddress})
		mockIPAddress.EXPECT().IP().Return(net.ParseIP("1.1.1.1"))
		mockIPAddress.EXPECT().IP().Return(net.ParseIP("8.8.8.8"))

		res, err := s.MachineIsRegisteredWithAPIServerDNS(&infrav1beta1.Machine{
			Addresses: []v1beta1.MachineAddress{
				{
					Type:    v1beta1.MachineInternalIP,
					Address: "1.1.1.1",
				},
				{
					Type:    v1beta1.MachineInternalIP,
					Address: "8.8.8.8",
				},
			},
		})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(res).To(BeTrue())
	})
}
