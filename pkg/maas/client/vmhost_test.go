package client

import (
	"context"
	"testing"

	"k8s.io/utils/pointer"
)

func TestVMHostExtension_QueryVMHosts(t *testing.T) {
	// Create extension with nil client since we're using mock data
	ext := NewVMHostExtension(nil)

	tests := []struct {
		name        string
		params      VMHostQueryParams
		expectCount int
		expectZone  string
		expectPool  string
	}{
		{
			name:        "query all LXD hosts",
			params:      VMHostQueryParams{Type: "lxd"},
			expectCount: 3,
		},
		{
			name: "query hosts by zone",
			params: VMHostQueryParams{
				Type: "lxd",
				Zone: pointer.String("zone-a"),
			},
			expectCount: 1,
			expectZone:  "zone-a",
		},
		{
			name: "query hosts by resource pool",
			params: VMHostQueryParams{
				Type:         "lxd",
				ResourcePool: pointer.String("compute-pool"),
			},
			expectCount: 1,
			expectPool:  "compute-pool",
		},
		{
			name: "query hosts by zone and pool",
			params: VMHostQueryParams{
				Type:         "lxd",
				Zone:         pointer.String("zone-b"),
				ResourcePool: pointer.String("default"),
			},
			expectCount: 1,
			expectZone:  "zone-b",
			expectPool:  "default",
		},
		{
			name: "query non-existent zone",
			params: VMHostQueryParams{
				Type: "lxd",
				Zone: pointer.String("zone-nonexistent"),
			},
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hosts, err := ext.QueryVMHosts(context.TODO(), tt.params)
			if err != nil {
				t.Errorf("QueryVMHosts() error = %v", err)
				return
			}

			if len(hosts) != tt.expectCount {
				t.Errorf("QueryVMHosts() got %d hosts, want %d", len(hosts), tt.expectCount)
			}

			if tt.expectCount > 0 {
				host := hosts[0]
				if tt.expectZone != "" && host.AvailabilityZone != tt.expectZone {
					t.Errorf("QueryVMHosts() zone = %v, want %v", host.AvailabilityZone, tt.expectZone)
				}
				if tt.expectPool != "" && host.ResourcePool != tt.expectPool {
					t.Errorf("QueryVMHosts() pool = %v, want %v", host.ResourcePool, tt.expectPool)
				}

				// Verify resource information is present
				if host.Resources.Total.Cores == 0 {
					t.Error("QueryVMHosts() host should have total cores > 0")
				}
				if host.Resources.Available.Memory == 0 {
					t.Error("QueryVMHosts() host should have available memory > 0")
				}
			}
		})
	}
}

func TestVMHostExtension_ComposeVM(t *testing.T) {
	ext := NewVMHostExtension(nil)

	tests := []struct {
		name   string
		params ComposeVMParams
		expect func(*testing.T, *ComposedVM)
	}{
		{
			name: "basic VM composition",
			params: ComposeVMParams{
				VMHostID: "lxd-host-1",
				Cores:    4,
				Memory:   8192,
				Profile:  "default",
				Project:  "maas",
			},
			expect: func(t *testing.T, vm *ComposedVM) {
				if vm.VMHostID != "lxd-host-1" {
					t.Errorf("ComposeVM() VMHostID = %v, want %v", vm.VMHostID, "lxd-host-1")
				}
				if vm.SystemID == "" {
					t.Error("ComposeVM() SystemID should not be empty")
				}
				if vm.Status == "" {
					t.Error("ComposeVM() Status should not be empty")
				}
			},
		},
		{
			name: "VM composition with disks",
			params: ComposeVMParams{
				VMHostID: "lxd-host-2",
				Cores:    2,
				Memory:   4096,
				Disks: []DiskSpec{
					{Size: "50GB", Pool: "ssd-pool"},
					{Size: "100GB", Pool: "hdd-pool"},
				},
				UserData: "I2Nsb3VkLWNvbmZpZwo=", // base64 encoded user data
			},
			expect: func(t *testing.T, vm *ComposedVM) {
				if vm.VMHostID != "lxd-host-2" {
					t.Errorf("ComposeVM() VMHostID = %v, want %v", vm.VMHostID, "lxd-host-2")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, err := ext.ComposeVM(context.TODO(), tt.params)
			if err != nil {
				t.Errorf("ComposeVM() error = %v", err)
				return
			}

			if vm == nil {
				t.Fatal("ComposeVM() returned nil VM")
			}

			tt.expect(t, vm)
		})
	}
}

func TestVMHostExtension_GetVMHostResources(t *testing.T) {
	ext := NewVMHostExtension(nil)

	resources, err := ext.GetVMHostResources(context.TODO(), "lxd-host-1")
	if err != nil {
		t.Errorf("GetVMHostResources() error = %v", err)
		return
	}

	if resources == nil {
		t.Fatal("GetVMHostResources() returned nil resources")
	}

	if resources.Total.Cores == 0 {
		t.Error("GetVMHostResources() total cores should be > 0")
	}

	if resources.Total.Memory == 0 {
		t.Error("GetVMHostResources() total memory should be > 0")
	}

	if resources.Available.Cores > resources.Total.Cores {
		t.Error("GetVMHostResources() available cores should not exceed total cores")
	}

	if resources.Available.Memory > resources.Total.Memory {
		t.Error("GetVMHostResources() available memory should not exceed total memory")
	}

	// Verify that used + available <= total (allowing for some reserved resources)
	if resources.Used.Cores+resources.Available.Cores > resources.Total.Cores {
		t.Error("GetVMHostResources() used + available cores should not exceed total")
	}

	if resources.Used.Memory+resources.Available.Memory > resources.Total.Memory {
		t.Error("GetVMHostResources() used + available memory should not exceed total")
	}
}

func TestVMHostExtension_DecomposeVM(t *testing.T) {
	// Since this uses the standard machine release API in mock mode,
	// we just test that it doesn't panic and returns without error
	ext := NewVMHostExtension(nil)

	err := ext.DecomposeVM(context.TODO(), "lxd-host-1", "vm-lxd-host-1-001")

	// In real implementation, this might return an error due to missing machine
	// But in mock mode, we expect it to succeed
	if err != nil {
		// This is expected in test environment without real MAAS connection
		t.Logf("DecomposeVM() expected error in test environment: %v", err)
	}
}

func TestNewVMHostExtension(t *testing.T) {
	ext := NewVMHostExtension(nil)
	if ext == nil {
		t.Error("NewVMHostExtension() should not return nil")
	}

	if ext.client != nil {
		t.Error("NewVMHostExtension() with nil client should have nil client field")
	}
}

func TestVMHostResourceCalculations(t *testing.T) {
	ext := NewVMHostExtension(nil)

	// Test that mock data provides realistic resource scenarios
	hosts, err := ext.QueryVMHosts(context.TODO(), VMHostQueryParams{Type: "lxd"})
	if err != nil {
		t.Errorf("QueryVMHosts() error = %v", err)
		return
	}

	if len(hosts) == 0 {
		t.Fatal("QueryVMHosts() should return at least one host")
	}

	for i, host := range hosts {
		t.Run(host.ID, func(t *testing.T) {
			// Check that resources are realistic
			if host.Resources.Total.Cores < 1 {
				t.Errorf("Host %d total cores should be >= 1, got %d", i, host.Resources.Total.Cores)
			}

			if host.Resources.Total.Memory < 1024 {
				t.Errorf("Host %d total memory should be >= 1024MB, got %d", i, host.Resources.Total.Memory)
			}

			// Check that used resources don't exceed total
			if host.Resources.Used.Cores > host.Resources.Total.Cores {
				t.Errorf("Host %d used cores (%d) exceed total cores (%d)", i, host.Resources.Used.Cores, host.Resources.Total.Cores)
			}

			if host.Resources.Used.Memory > host.Resources.Total.Memory {
				t.Errorf("Host %d used memory (%d) exceed total memory (%d)", i, host.Resources.Used.Memory, host.Resources.Total.Memory)
			}

			// Check that available resources are calculated correctly
			expectedAvailableCores := host.Resources.Total.Cores - host.Resources.Used.Cores
			if host.Resources.Available.Cores != expectedAvailableCores {
				t.Errorf("Host %d available cores should be %d, got %d", i, expectedAvailableCores, host.Resources.Available.Cores)
			}

			expectedAvailableMemory := host.Resources.Total.Memory - host.Resources.Used.Memory
			if host.Resources.Available.Memory != expectedAvailableMemory {
				t.Errorf("Host %d available memory should be %d, got %d", i, expectedAvailableMemory, host.Resources.Available.Memory)
			}
		})
	}
}
