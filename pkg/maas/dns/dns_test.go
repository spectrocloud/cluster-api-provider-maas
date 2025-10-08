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
		mockDNSResources := mockclientset.NewMockDNSResources(ctrl)
		mockDNSResourceBuilder := mockclientset.NewMockDNSResourceBuilder(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{dns: mockDNSResources},
		}
		mockDNSResources.EXPECT().List(context.Background(), gomock.Any()).Return(nil, nil)
		mockDNSResources.EXPECT().Builder().Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithFQDN(gomock.Any()).Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithAddressTTL("10").Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().WithIPAddresses(nil).Return(mockDNSResourceBuilder)
		mockDNSResourceBuilder.EXPECT().Create(gomock.Any())
		err := s.ReconcileDNS()

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.scope.GetDNSName()).To(ContainSubstring(cluster.Name))
		g.Expect(s.scope.GetDNSName()).To(ContainSubstring(maasCluster.Spec.DNSDomain))
	})

	t.Run("machine is registered", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResources := mockclientset.NewMockDNSResources(ctrl)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{dns: mockDNSResources},
		}
		mockDNSResources.EXPECT().List(context.Background(), gomock.Any()).Return([]maasclient.DNSResource{mockDNSResource}, nil)
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
			&fakeIPAddress{ip: net.ParseIP("8.8.8.8")},
		})

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

	t.Run("UpdateDNSAttachmentsWithResource - no change", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Current IPs match desired IPs -> expect no modify call
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
			&fakeIPAddress{ip: net.ParseIP("8.8.8.8")},
		})

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{"8.8.8.8", "1.1.1.1"})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeFalse(), "should not update when IPs match")
	})

	t.Run("UpdateDNSAttachmentsWithResource - with change", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		mockDNSResourceModifier := mockclientset.NewMockDNSResourceModifier(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Current IPs differ from desired -> expect modify call with sorted IPs
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
		})
		mockDNSResource.EXPECT().Modifier().Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().SetIPAddresses([]string{"1.1.1.1", "8.8.8.8"}).Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().Modify(gomock.Any()).Return(mockDNSResource, nil)

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{"8.8.8.8", "1.1.1.1"})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeTrue(), "should update when IPs differ")
	})

	t.Run("UpdateDNSAttachmentsWithResource - empty strings filtered", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Empty strings in input should be filtered
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
		})

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{"", "1.1.1.1", ""})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeFalse(), "empty strings should be filtered and result in no change")
	})

	t.Run("UpdateDNSAttachmentsWithResource - order insensitive", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Different order but same IPs -> no change
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("3.3.3.3")},
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
			&fakeIPAddress{ip: net.ParseIP("2.2.2.2")},
		})

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeFalse(), "order should not matter")
	})

	t.Run("UpdateDNSAttachmentsWithResource - empty to populated", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		mockDNSResourceModifier := mockclientset.NewMockDNSResourceModifier(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Empty current -> add IPs
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{})
		mockDNSResource.EXPECT().Modifier().Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().SetIPAddresses([]string{"1.1.1.1"}).Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().Modify(gomock.Any()).Return(mockDNSResource, nil)

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{"1.1.1.1"})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeTrue(), "should update when adding first IP")
	})

	t.Run("UpdateDNSAttachmentsWithResource - populated to empty", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockDNSResource := mockclientset.NewMockDNSResource(ctrl)
		mockDNSResourceModifier := mockclientset.NewMockDNSResourceModifier(ctrl)
		s := &Service{
			scope: &scope.ClusterScope{
				Logger:      log,
				Cluster:     cluster,
				MaasCluster: maasCluster,
			},
			maasClient: &fakeClientSet{},
		}

		// Remove all IPs
		mockDNSResource.EXPECT().IPAddresses().Return([]maasclient.IPAddress{
			&fakeIPAddress{ip: net.ParseIP("1.1.1.1")},
		})
		mockDNSResource.EXPECT().Modifier().Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().SetIPAddresses([]string{}).Return(mockDNSResourceModifier)
		mockDNSResourceModifier.EXPECT().Modify(gomock.Any()).Return(mockDNSResource, nil)

		updated, err := s.UpdateDNSAttachmentsWithResource(mockDNSResource, []string{})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated).To(BeTrue(), "should update when removing all IPs")
	})
}
