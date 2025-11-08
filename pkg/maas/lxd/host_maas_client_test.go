package lxd

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	mock_clientset "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/client/mock"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// ---- Fakes for maasclient interfaces ----

type fakeZone struct {
	name string
	id   int
}

func (z *fakeZone) Name() string        { return z.name }
func (z *fakeZone) ID() int             { return z.id }
func (z *fakeZone) Description() string { return "" }

type fakeResourcePool struct {
	name string
	id   int
}

func (p *fakeResourcePool) Name() string        { return p.name }
func (p *fakeResourcePool) ID() int             { return p.id }
func (p *fakeResourcePool) Description() string { return "" }

type fakeVMHost struct {
	name         string
	systemID     string
	hostSystemID string
	zone         *fakeZone
	pool         *fakeResourcePool
}

func (h *fakeVMHost) Get(context.Context) (maasclient.VMHost, error) { return h, nil }
func (h *fakeVMHost) Update(context.Context, maasclient.Params) (maasclient.VMHost, error) {
	return h, nil
}
func (h *fakeVMHost) Delete(context.Context) error        { return nil }
func (h *fakeVMHost) Composer() maasclient.VMComposer     { return nil }
func (h *fakeVMHost) Machines() maasclient.VMHostMachines { return nil }
func (h *fakeVMHost) SystemID() string                    { return h.systemID }
func (h *fakeVMHost) Name() string                        { return h.name }
func (h *fakeVMHost) Type() string                        { return "lxd" }
func (h *fakeVMHost) PowerAddress() string                { return "" }
func (h *fakeVMHost) HostSystemID() string                { return h.hostSystemID }
func (h *fakeVMHost) Zone() maasclient.Zone {
	if h.zone == nil {
		return nil
	}
	return h.zone
}
func (h *fakeVMHost) ResourcePool() maasclient.ResourcePool {
	if h.pool == nil {
		return nil
	}
	return h.pool
}
func (h *fakeVMHost) TotalCores() int                        { return 0 }
func (h *fakeVMHost) TotalMemory() int                       { return 0 }
func (h *fakeVMHost) UsedCores() int                         { return 0 }
func (h *fakeVMHost) UsedMemory() int                        { return 0 }
func (h *fakeVMHost) AvailableCores() int                    { return 0 }
func (h *fakeVMHost) AvailableMemory() int                   { return 0 }
func (h *fakeVMHost) Capabilities() []string                 { return nil }
func (h *fakeVMHost) Projects() []string                     { return nil }
func (h *fakeVMHost) StoragePools() []maasclient.StoragePool { return nil }

type fakeMachine struct {
	systemID   string
	powerState string
	state      string
	tags       []string
}

func (m *fakeMachine) Get(ctx context.Context) (maasclient.Machine, error) { return m, nil }
func (m *fakeMachine) Delete(context.Context) error                        { return nil }
func (m *fakeMachine) Releaser() maasclient.MachineReleaser                { return nil }
func (m *fakeMachine) Modifier() maasclient.MachineModifier                { return nil }
func (m *fakeMachine) Deployer() maasclient.MachineDeployer                { return nil }
func (m *fakeMachine) SystemID() string                                    { return m.systemID }
func (m *fakeMachine) FQDN() string                                        { return "" }
func (m *fakeMachine) Zone() maasclient.Zone                               { return nil }
func (m *fakeMachine) PowerState() string                                  { return m.powerState }
func (m *fakeMachine) PowerType() string                                   { return "" }
func (m *fakeMachine) Hostname() string                                    { return "" }
func (m *fakeMachine) IPAddresses() []net.IP                               { return nil }
func (m *fakeMachine) State() string                                       { return m.state }
func (m *fakeMachine) OSSystem() string                                    { return "" }
func (m *fakeMachine) DistroSeries() string                                { return "" }
func (m *fakeMachine) SwapSize() int                                       { return 0 }
func (m *fakeMachine) PowerManagerOn() maasclient.PowerManagerOn           { return nil }
func (m *fakeMachine) BootInterfaceID() string                             { return "" }
func (m *fakeMachine) TotalStorageGB() float64                             { return 0 }
func (m *fakeMachine) GetBootInterfaceType() string                        { return "" }
func (m *fakeMachine) ResourcePoolName() string                            { return "" }
func (m *fakeMachine) ZoneName() string                                    { return "" }
func (m *fakeMachine) BootInterfaceName() string                           { return "" }
func (m *fakeMachine) Tags() []string                                      { return m.tags }
func (m *fakeMachine) Parent() string                                      { return "" }

// ---- Tests ----

type fakeClient struct{ machines maasclient.Machines }

func (f fakeClient) Machines() maasclient.Machines { return f.machines }

func TestSelectLXDHost_StrictPrefersManaged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	machines := mock_clientset.NewMockMachines(ctrl)
	client := fakeClient{machines: machines}

	// Healthy, non-maintenance backing machines
	m1 := &fakeMachine{systemID: "H1", powerState: "on", state: "Deployed", tags: nil}
	m2 := &fakeMachine{systemID: "H2", powerState: "on", state: "Deployed", tags: nil}
	machines.EXPECT().Machine("H1").Return(m1).AnyTimes()
	machines.EXPECT().Machine("H2").Return(m2).AnyTimes()

	hosts := []maasclient.VMHost{
		&fakeVMHost{name: "lxd-host-a", systemID: "1", hostSystemID: "H1", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
		&fakeVMHost{name: "oob-host-b", systemID: "2", hostSystemID: "H2", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
	}

	selected, err := SelectLXDHostWithMaasClient(client, hosts, "z1", "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name() != "lxd-host-a" {
		t.Fatalf("expected managed host to be selected, got %s", selected.Name())
	}
}

func TestSelectLXDHost_StrictAllowsOOB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	machines := mock_clientset.NewMockMachines(ctrl)
	client := fakeClient{machines: machines}

	m := &fakeMachine{systemID: "H3", powerState: "on", state: "Deployed", tags: nil}
	machines.EXPECT().Machine("H3").Return(m).AnyTimes()

	hosts := []maasclient.VMHost{
		&fakeVMHost{name: "external-host", systemID: "3", hostSystemID: "H3", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
	}

	selected, err := SelectLXDHostWithMaasClient(client, hosts, "z1", "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name() != "external-host" {
		t.Fatalf("expected OOB host to be selected, got %s", selected.Name())
	}
}

func TestSelectLXDHost_StrictNoMatchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	machines := mock_clientset.NewMockMachines(ctrl)
	client := fakeClient{machines: machines}

	m := &fakeMachine{systemID: "H4", powerState: "on", state: "Deployed", tags: nil}
	machines.EXPECT().Machine("H4").Return(m).AnyTimes()

	hosts := []maasclient.VMHost{
		&fakeVMHost{name: "lxd-host-x", systemID: "4", hostSystemID: "H4", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
	}

	if _, err := SelectLXDHostWithMaasClient(client, hosts, "z2", "p1"); err == nil {
		t.Fatalf("expected error when no hosts match strict filters")
	}
}

func TestSelectLXDHost_SkipMaintenance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	machines := mock_clientset.NewMockMachines(ctrl)
	client := fakeClient{machines: machines}

	// First host is managed but under maintenance; second is healthy OOB
	m1 := &fakeMachine{systemID: "H5", powerState: "on", state: "Deployed", tags: []string{maintenance.TagHostMaintenance}}
	m2 := &fakeMachine{systemID: "H6", powerState: "on", state: "Deployed", tags: nil}
	machines.EXPECT().Machine("H5").Return(m1).AnyTimes()
	machines.EXPECT().Machine("H6").Return(m2).AnyTimes()

	hosts := []maasclient.VMHost{
		&fakeVMHost{name: "lxd-host-maint", systemID: "5", hostSystemID: "H5", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
		&fakeVMHost{name: "oob-healthy", systemID: "6", hostSystemID: "H6", zone: &fakeZone{name: "z1", id: 1}, pool: &fakeResourcePool{name: "p1", id: 1}},
	}

	selected, err := SelectLXDHostWithMaasClient(client, hosts, "z1", "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name() != "oob-healthy" {
		t.Fatalf("expected maintenance host to be skipped, got %s", selected.Name())
	}
}
