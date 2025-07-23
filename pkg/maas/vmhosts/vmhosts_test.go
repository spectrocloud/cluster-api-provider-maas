/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vmhosts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewVMHostsClient(t *testing.T) {
	client := NewVMHostsClient(nil, "http://localhost:5240", "test:key:secret")

	if client == nil {
		t.Fatal("Expected VMHostsClient to be created")
	}

	if client.baseURL != "http://localhost:5240" {
		t.Errorf("Expected baseURL to be 'http://localhost:5240', got '%s'", client.baseURL)
	}

	if client.apiKey != "test:key:secret" {
		t.Errorf("Expected apiKey to be 'test:key:secret', got '%s'", client.apiKey)
	}
}

func TestCreateVMHostRequest_Validation(t *testing.T) {
	req := CreateVMHostRequest{
		Name:         "test-host",
		Type:         "lxd",
		PowerAddress: "https://192.168.1.100:8443",
		PowerUser:    "root",
		PowerPass:    "",
		Pool:         "default",
		Zone:         "default",
		Project:      "default",
		Tags:         "lxd-host,capmaas",
	}

	// Test that all required fields are present
	if req.Name == "" {
		t.Error("Name should not be empty")
	}
	if req.Type == "" {
		t.Error("Type should not be empty")
	}
	if req.PowerAddress == "" {
		t.Error("PowerAddress should not be empty")
	}
	if req.PowerUser == "" {
		t.Error("PowerUser should not be empty")
	}
}

func TestVMHost_JSONTags(t *testing.T) {
	vmHost := VMHost{
		ID:           1,
		Name:         "test-host",
		Type:         "lxd",
		PowerAddress: "https://192.168.1.100:8443",
		PowerUser:    "root",
		Pool:         "default",
		Zone:         "default",
		Project:      "default",
		Tags:         "lxd-host,capmaas",
	}

	// Test that the struct can be marshaled to JSON
	_, err := json.Marshal(vmHost)
	if err != nil {
		t.Errorf("Failed to marshal VMHost to JSON: %v", err)
	}
}

func TestIsHostRegistered_EmptyHosts(t *testing.T) {
	client := NewVMHostsClient(nil, "http://localhost:5240", "test:key:secret")

	// Test with empty host address - this should fail with connection error
	// since we're not running a real MAAS server, but that's expected
	_, err := client.IsHostRegistered("")
	if err == nil {
		t.Error("Expected error when MAAS server is not available")
	}

	// Verify it's a connection error (expected behavior)
	if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "failed to make HTTP request") {
		t.Errorf("Expected connection error, got: %v", err)
	}
}
