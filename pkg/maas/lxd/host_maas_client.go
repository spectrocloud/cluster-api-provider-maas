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

package lxd

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/spectrocloud/maas-client-go/maasclient"
	"k8s.io/klog/v2/textlogger"
)

// HostConfig contains the configuration for setting up an LXD host
type HostConfig struct {
	NodeIP          string
	HostName        string
	MaasAPIKey      string
	MaasAPIEndpoint string
	StorageBackend  string
	StorageSize     string
	NetworkBridge   string
	Zone            string
	ResourcePool    string
	TrustPassword   string
}

// validateHostConfig validates the host configuration
func validateHostConfig(config HostConfig) error {
	if config.NodeIP == "" {
		return fmt.Errorf("node IP is required")
	}

	if config.MaasAPIKey == "" {
		return fmt.Errorf("MAAS API key is required")
	}

	if config.MaasAPIEndpoint == "" {
		return fmt.Errorf("MAAS API endpoint is required")
	}

	return nil
}

// SetupLXDHostWithMaasClient sets up an LXD host on a node using the official MAAS client
// This function assumes that LXD initialization is handled by the DaemonSet
// It only checks if the host is registered with MAAS and registers it if not
func SetupLXDHostWithMaasClient(config HostConfig) error {
	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("Setting up LXD host with official MAAS client", "node", config.NodeIP)

	// Validate configuration
	if err := validateHostConfig(config); err != nil {
		return fmt.Errorf("invalid host configuration: %w", err)
	}

	// Create MAAS client
	client := maasclient.NewAuthenticatedClientSet(config.MaasAPIEndpoint, config.MaasAPIKey)

	// Check if the host is already registered with MAAS
	isRegistered, err := isHostRegisteredWithMaasClient(client, config.NodeIP)
	if err != nil {
		return fmt.Errorf("failed to check if host is registered: %w", err)
	}

	if isRegistered {
		log.Info("LXD host is already registered with MAAS", "node", config.NodeIP)
		return nil
	}

	// Register the host with MAAS as a KVM host
	if err := registerWithMaasClient(client, config); err != nil {
		return fmt.Errorf("failed to register with MAAS: %w", err)
	}

	log.Info("Successfully set up LXD host", "node", config.NodeIP)
	return nil
}

// normalizeHost extracts the host part from a MAAS power_address or raw string
func normalizeHost(s string) string {
	if s == "" {
		return ""
	}
	// If there is no scheme, add one so url.Parse works.
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	h := u.Host
	if h == "" {
		h = u.Path // fallback when parse put everything into Path
	}
	if hp, _, err2 := net.SplitHostPort(h); err2 == nil {
		h = hp
	}
	return h
}

// isHostRegisteredWithMaasClient checks if a host is already registered with MAAS as a VM host
func isHostRegisteredWithMaasClient(client maasclient.ClientSetInterface, nodeIP string) (bool, error) {
	ctx := context.Background()

	vmHosts, err := client.VMHosts().List(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get VM hosts: %w", err)
	}

	wantName := fmt.Sprintf("lxd-host-%s", nodeIP)
	wantHost := normalizeHost(nodeIP)

	for _, host := range vmHosts {
		// Compare by name (our deterministic naming) or by normalized power address host component.
		if host.Name() == wantName {
			return true, nil
		}
		if normalizeHost(host.PowerAddress()) == wantHost {
			return true, nil
		}
	}
	return false, nil
}

// registerWithMaasClient registers a host with MAAS as a VM host
func registerWithMaasClient(client maasclient.ClientSetInterface, config HostConfig) error {
	ctx := context.Background()

	// Create registration parameters
	name := config.HostName
	if name == "" {
		name = fmt.Sprintf("lxd-host-%s", config.NodeIP)
	}
	params := maasclient.ParamsBuilder().
		Set("type", "lxd").
		Set("power_address", fmt.Sprintf("https://%s:8443", config.NodeIP)).
		Set("name", name)

	if config.Zone != "" {
		// Pass the zone name directly. MAAS API expects the zone name, not ID.
		params.Set("zone", config.Zone)
	}

	if config.ResourcePool != "" {
		// Pass pool name directly.
		params.Set("pool", config.ResourcePool)
	}

	if config.TrustPassword != "" {
		params.Set("password", config.TrustPassword)
	}

	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("register params", "zone", params.Values().Get("zone"), "pool", params.Values().Get("pool"), "name", name)

	// Register the host with MAAS
	_, err := client.VMHosts().Create(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to register host with MAAS: %w", err)
	}

	return nil
}

// GetAvailableLXDHostsWithMaasClient returns a list of available LXD hosts from MAAS
func GetAvailableLXDHostsWithMaasClient(apiKey, apiEndpoint string) ([]maasclient.VMHost, error) {
	// Create MAAS client
	client := maasclient.NewAuthenticatedClientSet(apiEndpoint, apiKey)

	// Get all VM hosts
	ctx := context.Background()
	vmHosts, err := client.VMHosts().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM hosts: %w", err)
	}

	return vmHosts, nil
}

// UnregisterLXDHostWithMaasClient removes a VM host registration from MAAS by matching name or power address
func UnregisterLXDHostWithMaasClient(apiKey, apiEndpoint, nodeIP string) error {
	client := maasclient.NewAuthenticatedClientSet(apiEndpoint, apiKey)

	ctx := context.Background()
	vmHosts, err := client.VMHosts().List(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get VM hosts: %w", err)
	}

	wantName := fmt.Sprintf("lxd-host-%s", nodeIP)
	wantHost := normalizeHost(nodeIP)

	for _, host := range vmHosts {
		if host.Name() == wantName || normalizeHost(host.PowerAddress()) == wantHost {
			// Found the host; delete it by system ID as required by the client
			if derr := client.VMHosts().VMHost(host.SystemID()).Delete(ctx); derr != nil {
				return fmt.Errorf("failed to delete VM host %s (id=%s): %w", host.Name(), host.SystemID(), derr)
			}
			log := textlogger.NewLogger(textlogger.NewConfig())
			log.Info("Successfully unregistered LXD host", "node", nodeIP, "id", host.SystemID(), "name", host.Name())
			return nil
		}
	}
	// Not found -> nothing to do
	return nil
}

// SelectLXDHostWithMaasClient selects an LXD host based on availability, AZ, and resource pool
func SelectLXDHostWithMaasClient(client maasclient.ClientSetInterface, hosts []maasclient.VMHost, az, resourcePool string) (maasclient.VMHost, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no LXD hosts available")
	}

	// First, try to find a host in the specified AZ and resource pool
	for _, host := range hosts {
		hostZone := ""
		if host.Zone() != nil {
			hostZone = host.Zone().Name()
		}

		hostPool := ""
		if host.ResourcePool() != nil {
			hostPool = host.ResourcePool().Name()
		}

		if (az == "" || hostZone == az) && (resourcePool == "" || hostPool == resourcePool) {

			// Check if the underlying host machine is deployed and powered on
			hostSystemID := host.HostSystemID()
			if hostSystemID != "" {
				// Check actual machine status using MAAS client
				ctx := context.Background()
				machine, err := client.Machines().Machine(hostSystemID).Get(ctx)
				if err != nil {
					log.Info("Failed to get machine details", "system-id", hostSystemID, "error", err.Error())
				} else {
					powerState := machine.PowerState()
					machineState := machine.State()
					isHealthy := powerState == "on" && machineState == "Deployed"

					if isHealthy {
						log.Info("Selected LXD host", "host-name", host.Name(), "host-id", host.SystemID())
						return host, nil
					}
				}
			}
			continue
		}
	}

	// If no host matches the AZ and resource pool, try to find a host in the specified AZ
	if resourcePool != "" {
		for _, host := range hosts {
			hostZone := ""
			if host.Zone() != nil {
				hostZone = host.Zone().Name()
			}

			if az == "" || hostZone == az {
				return host, nil
			}
		}
	}

	// If no host matches the AZ, try to find a host in the specified resource pool
	if az != "" {
		for _, host := range hosts {
			hostPool := ""
			if host.ResourcePool() != nil {
				hostPool = host.ResourcePool().Name()
			}

			if resourcePool == "" || hostPool == resourcePool {
				return host, nil
			}
		}
	}

	// If no host matches the criteria, return the first host
	return hosts[0], nil
}

// CreateLXDVMWithMaasClient creates a VM on an LXD host using MAAS API
func CreateLXDVMWithMaasClient(apiKey, apiEndpoint, vmHostID, vmName, vmCores, vmMemory, vmDisk, staticIP string) (string, error) {
	// Create MAAS client
	client := maasclient.NewAuthenticatedClientSet(apiEndpoint, apiKey)

	// Get the VM host
	vmHost := client.VMHosts().VMHost(vmHostID)

	// Create VM parameters
	params := maasclient.ParamsBuilder().
		Set("hostname", vmName).
		Set("cores", vmCores).
		Set("memory", vmMemory).
		Set("storage", vmDisk)

	if staticIP != "" {
		params.Set("interfaces", fmt.Sprintf("name=eth0,ip=%s", staticIP))
	}

	// Create the VM
	ctx := context.Background()
	machine, err := vmHost.Composer().Compose(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create VM: %w", err)
	}

	return machine.SystemID(), nil
}

// DeleteLXDVMWithMaasClient deletes a VM using MAAS API if it's an LXD VM
func DeleteLXDVMWithMaasClient(apiKey, apiEndpoint, systemID string) error {
	// Create MAAS client
	client := maasclient.NewAuthenticatedClientSet(apiEndpoint, apiKey)

	// Get the machine
	machine := client.Machines().Machine(systemID)
	ctx := context.Background()

	// Get machine details to check power type
	m, err := machine.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get machine %s: %w", systemID, err)
	}

	// Only delete if this is an LXD VM
	if m.PowerType() != "lxd" {
		return fmt.Errorf("machine %s is not an LXD VM (power_type: %s)", systemID, m.PowerType())
	}

	// Delete the LXD VM
	if err := machine.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete LXD VM: %w", err)
	}

	return nil
}
