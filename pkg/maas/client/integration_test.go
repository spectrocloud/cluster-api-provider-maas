package client

import (
	"context"
	"testing"

	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
)

func TestClientExtensions_VMHosts(t *testing.T) {
	ext := NewClientExtensions(nil)
	vmHosts := ext.VMHosts()

	if vmHosts == nil {
		t.Error("VMHosts() should not return nil")
	}

	// Test that we can call methods on the interface
	hosts, err := vmHosts.QueryVMHosts(context.TODO(), VMHostQueryParams{Type: "lxd"})
	if err != nil {
		t.Errorf("VMHosts().QueryVMHosts() error = %v", err)
	}

	if len(hosts) == 0 {
		t.Error("VMHosts().QueryVMHosts() should return at least one host")
	}
}

func TestLXDHostAdapter_ToLXDHost(t *testing.T) {
	vmHost := VMHost{
		ID:               "test-host-1",
		Name:             "Test Host 1",
		Type:             "lxd",
		AvailabilityZone: "zone-a",
		ResourcePool:     "default",
		Tags:             []string{"lxd", "compute"},
		Resources: VMHostResources{
			Total:     ResourceInfo{Cores: 16, Memory: 32768},
			Used:      ResourceInfo{Cores: 4, Memory: 8192},
			Available: ResourceInfo{Cores: 12, Memory: 24576},
		},
	}

	adapter := NewLXDHostAdapter(vmHost)
	lxdHost := adapter.ToLXDHost()

	// Verify all fields are correctly mapped
	if lxdHost.SystemID != vmHost.ID {
		t.Errorf("SystemID = %v, want %v", lxdHost.SystemID, vmHost.ID)
	}

	if lxdHost.Hostname != vmHost.Name {
		t.Errorf("Hostname = %v, want %v", lxdHost.Hostname, vmHost.Name)
	}

	if lxdHost.AvailabilityZone != vmHost.AvailabilityZone {
		t.Errorf("AvailabilityZone = %v, want %v", lxdHost.AvailabilityZone, vmHost.AvailabilityZone)
	}

	if lxdHost.ResourcePool != vmHost.ResourcePool {
		t.Errorf("ResourcePool = %v, want %v", lxdHost.ResourcePool, vmHost.ResourcePool)
	}

	// Verify resource mappings
	if lxdHost.Available.Cores != vmHost.Resources.Available.Cores {
		t.Errorf("Available.Cores = %v, want %v", lxdHost.Available.Cores, vmHost.Resources.Available.Cores)
	}

	if lxdHost.Available.Memory != vmHost.Resources.Available.Memory {
		t.Errorf("Available.Memory = %v, want %v", lxdHost.Available.Memory, vmHost.Resources.Available.Memory)
	}

	if lxdHost.Used.Cores != vmHost.Resources.Used.Cores {
		t.Errorf("Used.Cores = %v, want %v", lxdHost.Used.Cores, vmHost.Resources.Used.Cores)
	}

	if lxdHost.Used.Memory != vmHost.Resources.Used.Memory {
		t.Errorf("Used.Memory = %v, want %v", lxdHost.Used.Memory, vmHost.Resources.Used.Memory)
	}

	// Verify LXD capabilities are set
	if !lxdHost.LXDCapabilities.VMSupport {
		t.Error("LXDCapabilities.VMSupport should be true")
	}

	if len(lxdHost.LXDCapabilities.Projects) == 0 {
		t.Error("LXDCapabilities.Projects should not be empty")
	}
}

func TestVMHostsToLXDHosts(t *testing.T) {
	vmHosts := []VMHost{
		{
			ID:               "host-1",
			Name:             "Host 1",
			AvailabilityZone: "zone-a",
			Resources: VMHostResources{
				Total: ResourceInfo{Cores: 8, Memory: 16384},
			},
		},
		{
			ID:               "host-2",
			Name:             "Host 2",
			AvailabilityZone: "zone-b",
			Resources: VMHostResources{
				Total: ResourceInfo{Cores: 16, Memory: 32768},
			},
		},
	}

	lxdHosts := VMHostsToLXDHosts(vmHosts)

	if len(lxdHosts) != len(vmHosts) {
		t.Errorf("VMHostsToLXDHosts() length = %v, want %v", len(lxdHosts), len(vmHosts))
	}

	for i, lxdHost := range lxdHosts {
		if lxdHost.SystemID != vmHosts[i].ID {
			t.Errorf("lxdHosts[%d].SystemID = %v, want %v", i, lxdHost.SystemID, vmHosts[i].ID)
		}

		if lxdHost.Hostname != vmHosts[i].Name {
			t.Errorf("lxdHosts[%d].Hostname = %v, want %v", i, lxdHost.Hostname, vmHosts[i].Name)
		}
	}
}

func TestLXDVMSpecAdapter_ToComposeVMParams(t *testing.T) {
	vmSpec := &lxd.VMSpec{
		Cores:    4,
		Memory:   8192,
		UserData: "dGVzdC11c2VyZGF0YQ==", // base64 encoded
		Tags:     []string{"test", "vm"},
		Profile:  "maas",
		Project:  "test-project",
		Disks: []lxd.DiskSpec{
			{Size: "20GB", Pool: "default"},
			{Size: "100GB", Pool: "ssd-pool"},
		},
	}

	adapter := NewLXDVMSpecAdapter(vmSpec, "test-host-1")
	params := adapter.ToComposeVMParams()

	if params.VMHostID != "test-host-1" {
		t.Errorf("VMHostID = %v, want %v", params.VMHostID, "test-host-1")
	}

	if params.Cores != vmSpec.Cores {
		t.Errorf("Cores = %v, want %v", params.Cores, vmSpec.Cores)
	}

	if params.Memory != vmSpec.Memory {
		t.Errorf("Memory = %v, want %v", params.Memory, vmSpec.Memory)
	}

	if params.UserData != vmSpec.UserData {
		t.Errorf("UserData = %v, want %v", params.UserData, vmSpec.UserData)
	}

	if params.Profile != vmSpec.Profile {
		t.Errorf("Profile = %v, want %v", params.Profile, vmSpec.Profile)
	}

	if params.Project != vmSpec.Project {
		t.Errorf("Project = %v, want %v", params.Project, vmSpec.Project)
	}

	if len(params.Tags) != len(vmSpec.Tags) {
		t.Errorf("Tags length = %v, want %v", len(params.Tags), len(vmSpec.Tags))
	}

	if len(params.Disks) != len(vmSpec.Disks) {
		t.Errorf("Disks length = %v, want %v", len(params.Disks), len(vmSpec.Disks))
	}

	for i, disk := range params.Disks {
		if disk.Size != vmSpec.Disks[i].Size {
			t.Errorf("Disks[%d].Size = %v, want %v", i, disk.Size, vmSpec.Disks[i].Size)
		}

		if disk.Pool != vmSpec.Disks[i].Pool {
			t.Errorf("Disks[%d].Pool = %v, want %v", i, disk.Pool, vmSpec.Disks[i].Pool)
		}
	}
}

func TestComposedVMAdapter_ToLXDVMResult(t *testing.T) {
	composedVM := &ComposedVM{
		SystemID:         "vm-test-001",
		VMHostID:         "lxd-host-1",
		AvailabilityZone: "zone-a",
		Status:           "Allocating",
	}

	adapter := NewComposedVMAdapter(composedVM)

	tests := []struct {
		name             string
		project          string
		expectProject    string
		expectProviderID string
	}{
		{
			name:             "with project",
			project:          "test-project",
			expectProject:    "test-project",
			expectProviderID: "maas://zone-a/vm-test-001",
		},
		{
			name:             "without project",
			project:          "",
			expectProject:    "",
			expectProviderID: "maas://zone-a/vm-test-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.ToLXDVMResult(tt.project)

			if result.SystemID != composedVM.SystemID {
				t.Errorf("SystemID = %v, want %v", result.SystemID, composedVM.SystemID)
			}

			if result.HostID != composedVM.VMHostID {
				t.Errorf("HostID = %v, want %v", result.HostID, composedVM.VMHostID)
			}

			if result.FailureDomain != composedVM.AvailabilityZone {
				t.Errorf("FailureDomain = %v, want %v", result.FailureDomain, composedVM.AvailabilityZone)
			}

			if result.Project != tt.expectProject {
				t.Errorf("Project = %v, want %v", result.Project, tt.expectProject)
			}

			if result.ProviderID != tt.expectProviderID {
				t.Errorf("ProviderID = %v, want %v", result.ProviderID, tt.expectProviderID)
			}
		})
	}
}

func TestComposedVMAdapter_ToLXDVMResult_NoZone(t *testing.T) {
	composedVM := &ComposedVM{
		SystemID:         "vm-test-001",
		VMHostID:         "lxd-host-1",
		AvailabilityZone: "", // No zone
		Status:           "Allocating",
	}

	adapter := NewComposedVMAdapter(composedVM)
	result := adapter.ToLXDVMResult("test-project")

	expectedProviderID := "maas:///vm-test-001"
	if result.ProviderID != expectedProviderID {
		t.Errorf("ProviderID = %v, want %v", result.ProviderID, expectedProviderID)
	}
}

func TestDefaultClientExtensionFactory(t *testing.T) {
	factory := GetDefaultClientExtensionFactory()
	if factory == nil {
		t.Error("GetDefaultClientExtensionFactory() should not return nil")
	}

	extensions := factory.CreateExtensions(nil)
	if extensions == nil {
		t.Error("CreateExtensions() should not return nil")
	}

	// Test that the created extensions work
	vmHosts := extensions.VMHosts()
	if vmHosts == nil {
		t.Error("Created extensions VMHosts() should not return nil")
	}
}

func TestLXDVMSpecAdapter_EmptyDisks(t *testing.T) {
	vmSpec := &lxd.VMSpec{
		Cores:  2,
		Memory: 4096,
		Disks:  []lxd.DiskSpec{}, // Empty disks
	}

	adapter := NewLXDVMSpecAdapter(vmSpec, "test-host")
	params := adapter.ToComposeVMParams()

	if len(params.Disks) != 0 {
		t.Errorf("Empty disks should result in empty Disks slice, got %d disks", len(params.Disks))
	}
}

func TestLXDVMSpecAdapter_NilDisks(t *testing.T) {
	vmSpec := &lxd.VMSpec{
		Cores:  2,
		Memory: 4096,
		Disks:  nil, // Nil disks
	}

	adapter := NewLXDVMSpecAdapter(vmSpec, "test-host")
	params := adapter.ToComposeVMParams()

	if len(params.Disks) != 0 {
		t.Errorf("Nil disks should result in empty Disks slice, got %d disks", len(params.Disks))
	}
}
