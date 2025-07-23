package lxd

import (
	"context"
	"testing"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"k8s.io/utils/pointer"
)

func TestSelectOptimalHost(t *testing.T) {
	// Create a minimal service for testing
	service := &Service{}

	tests := []struct {
		name           string
		hosts          []LXDHost
		requiredCores  int
		requiredMemory int
		expectError    bool
		expectedHost   string
	}{
		{
			name:        "no hosts available",
			hosts:       []LXDHost{},
			expectError: true,
		},
		{
			name: "single suitable host",
			hosts: []LXDHost{
				{
					SystemID:  "host-1",
					Available: ResourceInfo{Cores: 8, Memory: 16384},
					Used:      ResourceInfo{Cores: 2, Memory: 4096},
				},
			},
			requiredCores:  4,
			requiredMemory: 8192,
			expectError:    false,
			expectedHost:   "host-1",
		},
		{
			name: "multiple hosts - select least utilized",
			hosts: []LXDHost{
				{
					SystemID:  "host-1",
					Available: ResourceInfo{Cores: 8, Memory: 16384},
					Used:      ResourceInfo{Cores: 6, Memory: 12288}, // High utilization
				},
				{
					SystemID:  "host-2",
					Available: ResourceInfo{Cores: 8, Memory: 16384},
					Used:      ResourceInfo{Cores: 2, Memory: 4096}, // Low utilization
				},
			},
			requiredCores:  2,
			requiredMemory: 2048,
			expectError:    false,
			expectedHost:   "host-2", // Should select less utilized host
		},
		{
			name: "insufficient resources",
			hosts: []LXDHost{
				{
					SystemID:  "host-1",
					Available: ResourceInfo{Cores: 2, Memory: 2048},
					Used:      ResourceInfo{Cores: 0, Memory: 0},
				},
			},
			requiredCores:  4,
			requiredMemory: 8192,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal machine scope with resource requirements
			machineScope := &scope.MachineScope{
				MaasMachine: &infrav1beta1.MaasMachine{
					Spec: infrav1beta1.MaasMachineSpec{
						MinCPU:        &tt.requiredCores,
						MinMemoryInMB: &tt.requiredMemory,
					},
				},
			}
			service.scope = machineScope

			result, err := service.SelectOptimalHost(context.TODO(), tt.hosts)

			if tt.expectError {
				if err == nil {
					t.Errorf("SelectOptimalHost() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("SelectOptimalHost() unexpected error: %v", err)
				return
			}

			if result.SystemID != tt.expectedHost {
				t.Errorf("SelectOptimalHost() = %v, want %v", result.SystemID, tt.expectedHost)
			}
		})
	}
}

func TestDistributeAcrossAZs(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name          string
		hosts         []LXDHost
		count         int
		expectedZones []string
		expectError   bool
	}{
		{
			name:        "no hosts available",
			hosts:       []LXDHost{},
			count:       2,
			expectError: true,
		},
		{
			name: "distribute across multiple zones",
			hosts: []LXDHost{
				{SystemID: "host-1", AvailabilityZone: "zone-a", Available: ResourceInfo{Cores: 8, Memory: 16384}},
				{SystemID: "host-2", AvailabilityZone: "zone-b", Available: ResourceInfo{Cores: 8, Memory: 16384}},
				{SystemID: "host-3", AvailabilityZone: "zone-c", Available: ResourceInfo{Cores: 8, Memory: 16384}},
			},
			count:         3,
			expectedZones: []string{"zone-a", "zone-b", "zone-c"},
			expectError:   false,
		},
		{
			name: "more VMs than zones - should distribute evenly",
			hosts: []LXDHost{
				{SystemID: "host-1", AvailabilityZone: "zone-a", Available: ResourceInfo{Cores: 16, Memory: 32768}},
				{SystemID: "host-2", AvailabilityZone: "zone-b", Available: ResourceInfo{Cores: 16, Memory: 32768}},
				{SystemID: "host-3", AvailabilityZone: "zone-a", Available: ResourceInfo{Cores: 16, Memory: 32768}},
			},
			count:       2,
			expectError: false,
		},
		{
			name: "single zone",
			hosts: []LXDHost{
				{SystemID: "host-1", AvailabilityZone: "zone-a", Available: ResourceInfo{Cores: 8, Memory: 16384}},
			},
			count:         1,
			expectedZones: []string{"zone-a"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal machine scope
			machineScope := &scope.MachineScope{
				MaasMachine: &infrav1beta1.MaasMachine{
					Spec: infrav1beta1.MaasMachineSpec{
						MinCPU:        pointer.Int(1),
						MinMemoryInMB: pointer.Int(1024),
					},
				},
			}
			service.scope = machineScope

			result, err := service.DistributeAcrossAZs(context.TODO(), tt.hosts, tt.count)

			if tt.expectError {
				if err == nil {
					t.Errorf("DistributeAcrossAZs() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DistributeAcrossAZs() unexpected error: %v", err)
				return
			}

			if len(result) != tt.count {
				t.Errorf("DistributeAcrossAZs() returned %d hosts, want %d", len(result), tt.count)
			}

			// Check that we got the expected zones if specified
			if len(tt.expectedZones) > 0 {
				resultZones := make(map[string]bool)
				for _, host := range result {
					resultZones[host.AvailabilityZone] = true
				}

				for _, expectedZone := range tt.expectedZones {
					if !resultZones[expectedZone] {
						t.Errorf("DistributeAcrossAZs() missing expected zone: %s", expectedZone)
					}
				}
			}
		})
	}
}

func TestNewService(t *testing.T) {
	// Skip this test since it requires MAAS environment variables
	t.Skip("Skipping TestNewService - requires MAAS environment variables")
}

func TestIsRetryableError_Integration(t *testing.T) {
	tests := []struct {
		name      string
		errorType LXDErrorType
		retryable bool
	}{
		{"host unavailable", LXDErrorHostUnavailable, true},
		{"insufficient resources", LXDErrorInsufficientResources, true},
		{"VM creation failed", LXDErrorVMCreationFailed, true},
		{"profile not found", LXDErrorProfileNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewLXDError(tt.errorType, "test error")
			if IsRetryableError(err) != tt.retryable {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.errorType, IsRetryableError(err), tt.retryable)
			}
		})
	}
}
