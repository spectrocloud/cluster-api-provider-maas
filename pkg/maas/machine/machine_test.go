package machine

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	mockclientset "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/client/mock"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
)

func TestMachine(t *testing.T) {
	log := klogr.New()
	cluster := &v1beta1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "a",
		},
	}

	//intPlaceholder := 400

	//maasMachine := &infrav1beta1.MaasMachine{
	//	TypeMeta:   v1.TypeMeta{},
	//	ObjectMeta: v1.ObjectMeta{},
	//	Spec: infrav1beta1.MaasMachineSpec{
	//		FailureDomain: pointer.String("zone1"),
	//		ResourcePool:  pointer.String("rp1"),
	//		MinCPU:        &intPlaceholder,
	//		MinMemoryInMB: &intPlaceholder,
	//		Image:         "custom-image",
	//	},
	//	Status: infrav1beta1.MaasMachineStatus{},
	//}

	t.Run("get machine with fqdn", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
		mockMachines := mockclientset.NewMockMachines(ctrl)
		mockMachine := mockclientset.NewMockMachine(ctrl)
		mockZone := mockclientset.NewMockZone(ctrl)

		s := &Service{
			scope: &scope.MachineScope{
				Logger:  log,
				Cluster: cluster,
			},
			maasClient: mockClientSetInterface,
		}

		mockClientSetInterface.EXPECT().Machines().Return(mockMachines)
		mockMachines.EXPECT().Machine("abc123").Return(mockMachine)
		mockMachine.EXPECT().Get(context.Background()).Return(mockMachine, nil)

		mockMachine.EXPECT().SystemID().Return("abc123")
		mockMachine.EXPECT().Hostname().Return("abc.hostanme")
		mockMachine.EXPECT().State().Return("Deployed")
		mockMachine.EXPECT().PowerState().Return("on")
		mockMachine.EXPECT().Zone().Return(mockZone)

		mockZone.EXPECT().Name().Return("zone1")

		mockMachine.EXPECT().FQDN().AnyTimes().Return("abc123.domain.local")
		mockMachine.EXPECT().IPAddresses().Return([]net.IP{net.ParseIP("1.2.3.4")})

		machine, err := s.GetMachine("abc123")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(machine).ToNot(BeNil())
		g.Expect(machine.Hostname).To(BeEquivalentTo("abc.hostanme"))
		g.Expect(machine.ID).To(BeEquivalentTo("abc123"))
		g.Expect(machine.Powered).To(BeTrue())
		g.Expect(machine.State).To(BeEquivalentTo("Deployed"))
		g.Expect(machine.AvailabilityZone).To(BeEquivalentTo("zone1"))
		g.Expect(machine.Addresses).To(ContainElements(v1beta1.MachineAddress{
			Type:    v1beta1.MachineExternalDNS,
			Address: "abc123.domain.local",
		}, v1beta1.MachineAddress{
			Type:    v1beta1.MachineExternalIP,
			Address: "1.2.3.4",
		}))
	})

	t.Run("release machine", func(t *testing.T) {
		g := NewGomegaWithT(t)
		ctrl := gomock.NewController(t)
		mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
		mockMachines := mockclientset.NewMockMachines(ctrl)
		mockMachine := mockclientset.NewMockMachine(ctrl)
		mockMachineReleaser := mockclientset.NewMockMachineReleaser(ctrl)

		s := &Service{
			scope: &scope.MachineScope{
				Logger:  log,
				Cluster: cluster,
			},
			maasClient: mockClientSetInterface,
		}

		mockClientSetInterface.EXPECT().Machines().Return(mockMachines)
		mockMachines.EXPECT().Machine("abc123").Return(mockMachine)
		mockMachine.EXPECT().Releaser().Return(mockMachineReleaser)
		mockMachineReleaser.EXPECT().Release(context.Background()).Return(mockMachine, nil)

		err := s.ReleaseMachine("abc123")
		g.Expect(err).ToNot(HaveOccurred())
	})

	//t.Run("deploy machine with success", func(t *testing.T) {
	//	g := NewGomegaWithT(t)
	//	ctrl := gomock.NewController(t)
	//	mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
	//	mockMachines := mockclientset.NewMockMachines(ctrl)
	//	mockMachine := mockclientset.NewMockMachine(ctrl)
	//	mockMachineAllocator := mockclientset.NewMockMachineAllocator(ctrl)
	//	mockMachineModifier := mockclientset.NewMockMachineModifier(ctrl)
	//	mockMachineDeployer := mockclientset.NewMockMachineDeployer(ctrl)
	//	mockZone := mockclientset.NewMockZone(ctrl)
	//
	//	s := &Service{
	//		scope: &scope.MachineScope{
	//			Logger:      log,
	//			Cluster:     cluster,
	//			MaasMachine: maasMachine,
	//		},
	//		maasClient: mockClientSetInterface,
	//	}
	//
	//	mockClientSetInterface.EXPECT().Machines().Return(mockMachines)
	//	mockMachines.EXPECT().Allocator().Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithZone("zone1").Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithResourcePool("rp1").Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithCPUCount(intPlaceholder).Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithMemory(intPlaceholder).Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().Allocate(context.Background()).Return(mockMachine, nil)
	//
	//	mockMachine.EXPECT().SystemID().AnyTimes().Return("abc123")
	//
	//	mockMachine.EXPECT().Modifier().Return(mockMachineModifier)
	//	mockMachineModifier.EXPECT().SetSwapSize(0).Return(mockMachineModifier)
	//	mockMachineModifier.EXPECT().Update(context.Background()).Return(mockMachine, nil)
	//
	//	mockMachine.EXPECT().Deployer().Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetUserData("userdata").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetDistroSeries("custom-image").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetOSSystem("custom").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().Deploy(context.Background()).Return(mockMachine, nil)
	//
	//	mockMachine.EXPECT().Hostname().Return("abc.hostanme")
	//	mockMachine.EXPECT().State().Return("Deployed")
	//	mockMachine.EXPECT().PowerState().Return("on")
	//	mockMachine.EXPECT().Zone().Return(mockZone)
	//
	//	mockZone.EXPECT().Name().Return("zone1")
	//
	//	mockMachine.EXPECT().FQDN().AnyTimes().Return("abc123.domain.local")
	//	mockMachine.EXPECT().IPAddresses().Return([]net.IP{net.ParseIP("1.2.3.4")})
	//	//mockMachines.EXPECT().Machine("abc123").Return(mockMachine)
	//	//mockMachine.EXPECT().Releaser().Return(mockMachineReleaser)
	//	//mockMachineReleaser.EXPECT().Release(context.Background()).Return(mockMachine, nil)
	//
	//	machine, err := s.DeployMachine("userdata")
	//	g.Expect(err).ToNot(HaveOccurred())
	//	g.Expect(machine).ToNot(BeNil())
	//	g.Expect(machine.Hostname).To(BeEquivalentTo("abc.hostanme"))
	//	g.Expect(machine.ID).To(BeEquivalentTo("abc123"))
	//	g.Expect(machine.Powered).To(BeTrue())
	//	g.Expect(machine.State).To(BeEquivalentTo("Deployed"))
	//	g.Expect(machine.AvailabilityZone).To(BeEquivalentTo("zone1"))
	//	g.Expect(machine.Addresses).To(ContainElements(v1beta1.MachineAddress{
	//		Type:    v1beta1.MachineExternalDNS,
	//		Address: "abc123.domain.local",
	//	}, v1beta1.MachineAddress{
	//		Type:    v1beta1.MachineExternalIP,
	//		Address: "1.2.3.4",
	//	}))
	//})
	//
	//t.Run("deploy machine with failure should release machine", func(t *testing.T) {
	//	g := NewGomegaWithT(t)
	//	ctrl := gomock.NewController(t)
	//	mockClientSetInterface := mockclientset.NewMockClientSetInterface(ctrl)
	//	mockMachines := mockclientset.NewMockMachines(ctrl)
	//	mockMachine := mockclientset.NewMockMachine(ctrl)
	//	mockMachineAllocator := mockclientset.NewMockMachineAllocator(ctrl)
	//	mockMachineModifier := mockclientset.NewMockMachineModifier(ctrl)
	//	mockMachineDeployer := mockclientset.NewMockMachineDeployer(ctrl)
	//	mockMachineReleaser := mockclientset.NewMockMachineReleaser(ctrl)
	//
	//	s := &Service{
	//		scope: &scope.MachineScope{
	//			Logger:      log,
	//			Cluster:     cluster,
	//			MaasMachine: maasMachine,
	//		},
	//		maasClient: mockClientSetInterface,
	//	}
	//
	//	mockClientSetInterface.EXPECT().Machines().Return(mockMachines)
	//	mockMachines.EXPECT().Allocator().Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithZone("zone1").Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithResourcePool("rp1").Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithCPUCount(intPlaceholder).Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().WithMemory(intPlaceholder).Return(mockMachineAllocator)
	//	mockMachineAllocator.EXPECT().Allocate(context.Background()).Return(mockMachine, nil)
	//
	//	mockMachine.EXPECT().SystemID().AnyTimes().Return("abc123")
	//
	//	mockMachine.EXPECT().Modifier().Return(mockMachineModifier)
	//	mockMachineModifier.EXPECT().SetSwapSize(0).Return(mockMachineModifier)
	//	mockMachineModifier.EXPECT().Update(context.Background()).Return(mockMachine, nil)
	//
	//	mockMachine.EXPECT().Deployer().Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetUserData("userdata").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetDistroSeries("custom-image").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().SetOSSystem("custom").Return(mockMachineDeployer)
	//	mockMachineDeployer.EXPECT().Deploy(context.Background()).Return(nil, errors.New("mock failure deploy"))
	//
	//	mockMachine.EXPECT().Releaser().Return(mockMachineReleaser)
	//	mockMachineReleaser.EXPECT().Release(context.Background()).Return(mockMachine, nil)
	//
	//	machine, err := s.DeployMachine("userdata")
	//	g.Expect(err).To(HaveOccurred())
	//	g.Expect(machine).To(BeNil())
	//})
}
