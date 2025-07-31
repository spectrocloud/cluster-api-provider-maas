// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package lxd

// import (
// 	"fmt"
// 	"net/url"
// 	"strings"

// 	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/vmhosts"
// 	"k8s.io/klog/v2/textlogger"
// )

// // HostConfig contains the configuration for setting up an LXD host
// type HostConfig struct {
// 	NodeIP          string
// 	MaasAPIKey      string
// 	MaasAPIEndpoint string
// 	StorageBackend  string
// 	StorageSize     string
// 	NetworkBridge   string
// 	Zone            string
// 	ResourcePool    string
// 	TrustPassword   string
// }

// // validateHostConfig validates the host configuration
// func validateHostConfig(config HostConfig) error {
// 	if config.NodeIP == "" {
// 		return fmt.Errorf("node IP is required")
// 	}

// 	if config.MaasAPIKey == "" {
// 		return fmt.Errorf("MAAS API key is required")
// 	}

// 	if config.MaasAPIEndpoint == "" {
// 		return fmt.Errorf("MAAS API endpoint is required")
// 	}

// 	return nil
// }

// // SetupLXDHost sets up an LXD host on a node using MAAS API
// // This function now assumes that LXD initialization is handled by the DaemonSet
// // It only checks if the host is registered with MAAS and registers it if not
// func SetupLXDHost(config HostConfig) error {
// 	log := textlogger.NewLogger(textlogger.NewConfig())
// 	log.Info("Setting up LXD host", "node", config.NodeIP)

// 	// Validate configuration
// 	if err := validateHostConfig(config); err != nil {
// 		return fmt.Errorf("invalid host configuration: %w", err)
// 	}

// 	// Check if the host is already registered with MAAS
// 	isRegistered, err := isHostRegistered(config.MaasAPIKey, config.MaasAPIEndpoint, config.NodeIP)
// 	if err != nil {
// 		return fmt.Errorf("failed to check if host is registered: %w", err)
// 	}

// 	if isRegistered {
// 		log.Info("LXD host is already registered with MAAS", "node", config.NodeIP)
// 		return nil
// 	}

// 	// Register the host with MAAS as a KVM host
// 	if err := registerWithMAAS(config); err != nil {
// 		return fmt.Errorf("failed to register with MAAS: %w", err)
// 	}

// 	log.Info("Successfully set up LXD host", "node", config.NodeIP)
// 	return nil
// }

// // isHostRegistered checks if a host is already registered with MAAS as a VM host
// func isHostRegistered(apiKey, apiEndpoint, nodeIP string) (bool, error) {
// 	client := vmhosts.NewClient(apiKey, apiEndpoint)

// 	hosts, err := client.GetVMHosts()
// 	if err != nil {
// 		return false, fmt.Errorf("failed to get VM hosts: %w", err)
// 	}

// 	for _, host := range hosts {
// 		// Check if the host's power address contains the node IP
// 		if strings.Contains(host.PowerAddress, nodeIP) {
// 			return true, nil
// 		}
// 	}

// 	return false, nil
// }

// // registerWithMAAS registers a host with MAAS as a VM host
// func registerWithMAAS(config HostConfig) error {
// 	client := vmhosts.NewClient(config.MaasAPIKey, config.MaasAPIEndpoint)

// 	// Create registration parameters
// 	params := url.Values{}
// 	params.Add("type", "lxd")
// 	params.Add("power_address", fmt.Sprintf("https://%s:8443", config.NodeIP))
// 	params.Add("name", fmt.Sprintf("lxd-host-%s", config.NodeIP))

// 	if config.Zone != "" {
// 		params.Add("zone", config.Zone)
// 	}

// 	if config.ResourcePool != "" {
// 		params.Add("pool", config.ResourcePool)
// 	}

// 	// Register the host with MAAS
// 	_, err := client.CreateVMHostWithParams(params)
// 	if err != nil {
// 		return fmt.Errorf("failed to register host with MAAS: %w", err)
// 	}

// 	return nil
// }

// // GetAvailableLXDHosts returns a list of available LXD hosts from MAAS
// func GetAvailableLXDHosts(apiKey, apiEndpoint string) ([]vmhosts.VMHost, error) {
// 	client := vmhosts.NewClient(apiKey, apiEndpoint)

// 	hosts, err := client.GetVMHosts()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get VM hosts: %w", err)
// 	}

// 	return hosts, nil
// }

// // SelectLXDHost selects an LXD host based on availability, AZ, and resource pool
// func SelectLXDHost(hosts []vmhosts.VMHost, az, resourcePool string) (*vmhosts.VMHost, error) {
// 	if len(hosts) == 0 {
// 		return nil, fmt.Errorf("no LXD hosts available")
// 	}

// 	// First, try to find a host in the specified AZ and resource pool
// 	for _, host := range hosts {
// 		if (az == "" || host.Zone == az) && (resourcePool == "" || host.Pool == resourcePool) {
// 			return &host, nil
// 		}
// 	}

// 	// If no host matches the AZ and resource pool, try to find a host in the specified AZ
// 	if resourcePool != "" {
// 		for _, host := range hosts {
// 			if az == "" || host.Zone == az {
// 				return &host, nil
// 			}
// 		}
// 	}

// 	// If no host matches the AZ, try to find a host in the specified resource pool
// 	if az != "" {
// 		for _, host := range hosts {
// 			if resourcePool == "" || host.Pool == resourcePool {
// 				return &host, nil
// 			}
// 		}
// 	}

// 	// If no host matches the criteria, return the first host
// 	return &hosts[0], nil
// }

// // CreateLXDVM creates a VM on an LXD host using MAAS API
// func CreateLXDVM(apiKey, apiEndpoint, vmHostID, vmName, vmCores, vmMemory, vmDisk, staticIP string) (string, error) {
// 	client := vmhosts.NewClient(apiKey, apiEndpoint)

// 	// Create VM parameters
// 	params := url.Values{}
// 	params.Add("hostname", vmName)
// 	params.Add("cores", vmCores)
// 	params.Add("memory", vmMemory)
// 	params.Add("storage", vmDisk)

// 	if staticIP != "" {
// 		params.Add("interfaces", fmt.Sprintf("name=eth0,ip=%s", staticIP))
// 	}

// 	// Create the VM
// 	machine, err := client.ComposeVM(vmHostID, params)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create VM: %w", err)
// 	}

// 	return machine.SystemID, nil
// }

// // DeleteLXDVM deletes a VM using MAAS API
// func DeleteLXDVM(apiKey, apiEndpoint, systemID string) error {
// 	client := vmhosts.NewClient(apiKey, apiEndpoint)

// 	// Delete the VM
// 	err := client.DeleteMachine(systemID)
// 	if err != nil {
// 		return fmt.Errorf("failed to delete VM: %w", err)
// 	}

// 	return nil
// }
