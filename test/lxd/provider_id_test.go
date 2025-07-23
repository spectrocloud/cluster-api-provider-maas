// +build integration

package lxd

import (
	"testing"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	infrautil "github.com/spectrocloud/cluster-api-provider-maas/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// TestLXDProviderIDGeneration tests LXD provider ID generation and parsing
func TestLXDProviderIDGeneration(t *testing.T) {
	tests := []struct {
		name             string
		systemID         string
		availabilityZone string
		expectedFormat   string
		shouldParseLXD   bool
	}{
		{
			name:             "LXD VM with zone",
			systemID:         "test-vm-123",
			availabilityZone: "zone-a",
			expectedFormat:   "maas-lxd:///zone-a/test-vm-123/vm-name-123",
			shouldParseLXD:   true,
		},
		{
			name:             "LXD VM without zone",
			systemID:         "test-vm-456",
			availabilityZone: "",
			expectedFormat:   "maas-lxd:////test-vm-456/vm-name-123",
			shouldParseLXD:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test objects
			maasMachine := &infrav1beta1.MaasMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: infrav1beta1.MaasMachineSpec{
					ProvisioningMode: infrav1beta1.ProvisioningModeLXD,
				},
			}

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
			}

			machine := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
			}

			// Create machine scope
			machineScope := &scope.MachineScope{
				MaasMachine: maasMachine,
				Machine:     machine,
				Cluster:     cluster,
			}

			// Test LXD provider ID generation
			machineScope.SetLXDProviderID("vm-name-123", tc.systemID, tc.availabilityZone)

			// Verify provider ID format
			providerID := machineScope.GetProviderID()
			if providerID != tc.expectedFormat {
				t.Errorf("Provider ID = %s, want %s", providerID, tc.expectedFormat)
			}

			// Test provider ID type detection
			// Test provider ID type detection using IsLXDProviderID
			if !machineScope.IsLXDProviderID() {
				t.Error("IsLXDProviderID() should return true")
			}

			// Provider ID methods tested above

			// Test provider ID parsing
			parsed, err := infrautil.NewProviderID(providerID)
			if err != nil {
				t.Errorf("Failed to parse provider ID: %v", err)
				return
			}

			if parsed.IsLXD() != tc.shouldParseLXD {
				t.Errorf("IsLXD() = %v, want %v", parsed.IsLXD(), tc.shouldParseLXD)
			}

			if parsed.ID() != tc.systemID {
				t.Errorf("Parsed system ID = %s, want %s", parsed.ID(), tc.systemID)
			}

			// Provisioning type test removed as method doesn't exist

			// Test LXD-specific provider ID parsing
			zone, hostSystemID, vmName, err := infrautil.ParseLXDProviderID(providerID)
			if err != nil {
				t.Errorf("Failed to parse LXD provider ID: %v", err)
				return
			}

			if hostSystemID != tc.systemID {
				t.Errorf("Parsed host system ID = %s, want %s", hostSystemID, tc.systemID)
			}

			if zone != tc.availabilityZone {
				t.Errorf("Parsed zone = %s, want %s", zone, tc.availabilityZone)
			}

			// vmName should be "vm-name-123" as set above
			if vmName != "vm-name-123" {
				t.Errorf("Parsed VM name = %s, want vm-name-123", vmName)
			}
		})
	}
}

// TestBareMetalProviderIDGeneration tests bare metal provider ID generation
func TestBareMetalProviderIDGeneration(t *testing.T) {
	// Create test objects for bare metal
	maasMachine := &infrav1beta1.MaasMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
		Spec: infrav1beta1.MaasMachineSpec{
			ProvisioningMode: infrav1beta1.ProvisioningModeBaremetal,
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	// Create machine scope
	machineScope := &scope.MachineScope{
		MaasMachine: maasMachine,
		Machine:     machine,
		Cluster:     cluster,
	}

	// Test bare metal provider ID generation
	systemID := "test-machine-789"
	availabilityZone := "zone-b"
	expectedFormat := "maas:///zone-b/test-machine-789"

	machineScope.SetProviderID(systemID, availabilityZone)

	// Verify provider ID format
	providerID := machineScope.GetProviderID()
	if providerID != expectedFormat {
		t.Errorf("Provider ID = %s, want %s", providerID, expectedFormat)
	}

	// Test provider ID type detection
	if machineScope.IsLXDProviderID() {
		t.Error("IsLXDProviderID() should return false for bare metal provider ID")
	}

	// Test provider ID parsing
	parsed, err := infrautil.NewProviderID(providerID)
	if err != nil {
		t.Errorf("Failed to parse provider ID: %v", err)
		return
	}

	if parsed.IsLXD() {
		t.Error("IsLXD should return false for bare metal provider ID")
	}

	if parsed.ID() != systemID {
		t.Errorf("Parsed system ID = %s, want %s", parsed.ID(), systemID)
	}

	// Provisioning type test removed as method doesn't exist
}

// TestProvisioningModeDetection tests the provisioning mode detection methods
func TestProvisioningModeDetection(t *testing.T) {
	tests := []struct {
		name             string
		provisioningMode *infrav1beta1.ProvisioningMode
		expectLXD        bool
	}{
		{
			name:             "LXD provisioning mode",
			provisioningMode: &[]infrav1beta1.ProvisioningMode{infrav1beta1.ProvisioningModeLXD}[0],
			expectLXD:        true,
		},
		{
			name:             "Bare metal provisioning mode",
			provisioningMode: &[]infrav1beta1.ProvisioningMode{infrav1beta1.ProvisioningModeBaremetal}[0],
			expectLXD:        false,
		},
		{
			name:             "Nil provisioning mode (defaults to bare metal)",
			provisioningMode: nil,
			expectLXD:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maasMachine := &infrav1beta1.MaasMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: infrav1beta1.MaasMachineSpec{
					ProvisioningMode: func() infrav1beta1.ProvisioningMode {
						if tc.provisioningMode != nil {
							return *tc.provisioningMode
						}
						return infrav1beta1.ProvisioningModeBaremetal
					}(),
				},
			}

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
			}

			machine := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
			}

			// Create machine scope
			machineScope := &scope.MachineScope{
				MaasMachine: maasMachine,
				Machine:     machine,
				Cluster:     cluster,
			}

			// Test provisioning mode detection
			isLXD := machineScope.IsLXDProvisioning()
			if isLXD != tc.expectLXD {
				t.Errorf("IsLXDProvisioning() = %v, want %v", isLXD, tc.expectLXD)
			}
		})
	}
}

// TestLXDMetadataManagement tests LXD-specific metadata management
func TestLXDMetadataManagement(t *testing.T) {
	maasMachine := &infrav1beta1.MaasMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
		Spec: infrav1beta1.MaasMachineSpec{
			ProvisioningMode: infrav1beta1.ProvisioningModeLXD,
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	// Create machine scope
	machineScope := &scope.MachineScope{
		MaasMachine: maasMachine,
		Machine:     machine,
		Cluster:     cluster,
	}

	// Test LXD status management
	testHostSystemID := "lxd-host-123"
	testVMName := "vm-name-456"
	testHostAddress := "192.168.1.100"
	
	machineScope.SetLXDStatus(testHostSystemID, testVMName, testHostAddress, nil)

	lxdStatus := machineScope.GetLXDStatus()
	if lxdStatus == nil {
		t.Error("LXD status should not be nil")
		return
	}

	if lxdStatus.HostSystemID == nil || *lxdStatus.HostSystemID != testHostSystemID {
		t.Errorf("LXD Host System ID = %v, want %s", lxdStatus.HostSystemID, testHostSystemID)
	}

	if lxdStatus.VMName == nil || *lxdStatus.VMName != testVMName {
		t.Errorf("LXD VM Name = %v, want %s", lxdStatus.VMName, testVMName)
	}

	// Test retrieving individual values
	retrievedHostID := machineScope.GetLXDHostSystemID()
	if retrievedHostID != testHostSystemID {
		t.Errorf("GetLXDHostSystemID() = %s, want %s", retrievedHostID, testHostSystemID)
	}

	retrievedVMName := machineScope.GetLXDVMName()
	if retrievedVMName != testVMName {
		t.Errorf("GetLXDVMName() = %s, want %s", retrievedVMName, testVMName)
	}
}
