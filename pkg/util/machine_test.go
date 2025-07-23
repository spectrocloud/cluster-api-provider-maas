package util

import (
	"testing"
)

func TestProviderIDParsing(t *testing.T) {
	tests := []struct {
		name        string
		providerID  string
		expectError bool
		expectedID  string
		expectedCP  string
		isLXD       bool
		provType    string
	}{
		{
			name:        "Valid bare metal provider ID",
			providerID:  "maas:///zone-a/machine-123",
			expectError: false,
			expectedID:  "machine-123",
			expectedCP:  "maas",
			isLXD:       false,
			provType:    "bare-metal",
		},
		{
			name:        "Valid LXD provider ID with zone",
			providerID:  "maas-lxd:///zone-a/host-123/vm-456",
			expectError: false,
			expectedID:  "host-123",
			expectedCP:  "maas-lxd",
			isLXD:       true,
			provType:    "lxd",
		},
		{
			name:        "Valid LXD provider ID without zone",
			providerID:  "maas-lxd:////host-789/vm-789",
			expectError: false,
			expectedID:  "host-789",
			expectedCP:  "maas-lxd",
			isLXD:       true,
			provType:    "lxd",
		},
		{
			name:        "Invalid provider ID format",
			providerID:  "invalid-format",
			expectError: true,
		},
		{
			name:        "Empty provider ID",
			providerID:  "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := NewProviderID(tc.providerID)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if parsed.ID() != tc.expectedID {
				t.Errorf("ID() = %s, want %s", parsed.ID(), tc.expectedID)
			}

			if parsed.CloudProvider() != tc.expectedCP {
				t.Errorf("CloudProvider() = %s, want %s", parsed.CloudProvider(), tc.expectedCP)
			}

			if parsed.IsLXD() != tc.isLXD {
				t.Errorf("IsLXD() = %v, want %v", parsed.IsLXD(), tc.isLXD)
			}

			// Remove GetProvisioningType test as this method doesn't exist

			// Test index key
			if parsed.IndexKey() != tc.providerID {
				t.Errorf("IndexKey() = %s, want %s", parsed.IndexKey(), tc.providerID)
			}
		})
	}
}

func TestParseLXDProviderID(t *testing.T) {
	tests := []struct {
		name         string
		providerID   string
		expectedVMID string
		expectedZone string
		expectedHost string
		expectError  bool
	}{
		{
			name:         "LXD provider ID with zone",
			providerID:   "maas-lxd:///zone-a/host-123/vm-123",
			expectedVMID: "vm-123",
			expectedZone: "zone-a",
			expectedHost: "host-123",
			expectError:  false,
		},
		{
			name:         "LXD provider ID with default zone",
			providerID:   "maas-lxd:///default/host-456/vm-456",
			expectedVMID: "vm-456",
			expectedZone: "default",
			expectedHost: "host-456",
			expectError:  false,
		},
		{
			name:         "LXD provider ID with empty zone",
			providerID:   "maas-lxd:////host-789/vm-789",
			expectedVMID: "vm-789",
			expectedZone: "",
			expectedHost: "host-789",
			expectError:  false,
		},
		{
			name:        "Non-LXD provider ID",
			providerID:  "maas:///zone-a/machine-123",
			expectError: true,
		},
		{
			name:        "Invalid format",
			providerID:  "invalid-format",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			zone, hostSystemID, vmName, err := ParseLXDProviderID(tc.providerID)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if vmName != tc.expectedVMID {
				t.Errorf("VM Name = %s, want %s", vmName, tc.expectedVMID)
			}

			if zone != tc.expectedZone {
				t.Errorf("Zone = %s, want %s", zone, tc.expectedZone)
			}

			if hostSystemID != tc.expectedHost {
				t.Errorf("Host System ID = %s, want %s", hostSystemID, tc.expectedHost)
			}
		})
	}
}
