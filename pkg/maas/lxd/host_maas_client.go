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
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
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

	// Check if the host is already registered with MAAS (by systemID, desired name, or power address)
	hn := strings.TrimSpace(config.HostName)
	desiredName := fmt.Sprintf("lxd-host-%s", hn)
	if hn == "" {
		desiredName = fmt.Sprintf("lxd-host-%s", config.NodeIP)
	}
	isRegistered, err := isHostRegisteredWithMaasClientAdvanced(client, "", desiredName, config.NodeIP)
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
// isHostRegisteredWithMaasClientAdvanced returns true if a VM host exists matching systemID, desired name, or power address host
func isHostRegisteredWithMaasClientAdvanced(client maasclient.ClientSetInterface, systemID, desiredName, nodeIP string) (bool, error) {
	ctx := context.Background()

	vmHosts, err := client.VMHosts().List(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get VM hosts: %w", err)
	}

	wantName := strings.TrimSpace(desiredName)
	wantHost := normalizeHost(strings.TrimSpace(nodeIP))

	for _, host := range vmHosts {
		// 1) Prefer exact hostSystemID match when provided
		if strings.TrimSpace(systemID) != "" && host.HostSystemID() == strings.TrimSpace(systemID) {
			return true, nil
		}
		// 2) Compare by desired name
		if wantName != "" && host.Name() == wantName {
			return true, nil
		}
		// 3) Legacy power address host match
		if wantHost != "" && normalizeHost(host.PowerAddress()) == wantHost {
			return true, nil
		}
	}
	return false, nil
}

// registerWithMaasClient registers a host with MAAS as a VM host
func registerWithMaasClient(client maasclient.ClientSetInterface, config HostConfig) error {
	ctx := context.Background()

	// Create registration parameters
	name := strings.TrimSpace(config.HostName)
	if name != "" {
		name = fmt.Sprintf("lxd-host-%s", name)
	} else {
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

// UnregisterLXDHostByNameWithMaasClient removes a VM host registration from MAAS by matching the exact host name
func UnregisterLXDHostByNameWithMaasClient(apiKey, apiEndpoint, hostName string) error {
	client := maasclient.NewAuthenticatedClientSet(apiEndpoint, apiKey)

	ctx := context.Background()
	vmHosts, err := client.VMHosts().List(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get VM hosts: %w", err)
	}

	for _, host := range vmHosts {
		if host.Name() == hostName {
			if derr := client.VMHosts().VMHost(host.SystemID()).Delete(ctx); derr != nil {
				return fmt.Errorf("failed to delete VM host %s (id=%s): %w", host.Name(), host.SystemID(), derr)
			}
			log := textlogger.NewLogger(textlogger.NewConfig())
			log.Info("Successfully unregistered LXD host", "name", hostName, "id", host.SystemID())
			return nil
		}
	}
	return nil
}

// isHostUnderMaintenance checks if a host has maintenance tags
func isHostUnderMaintenance(client machineGetter, hostSystemID string, log logr.Logger) bool {
	if hostSystemID == "" {
		return false
	}

	ctx := context.Background()
	machine, err := client.Machines().Machine(hostSystemID).Get(ctx)
	if err != nil {
		log.Info("Failed to get machine details for maintenance check", "system-id", hostSystemID, "error", err.Error())
		return false // Assume not under maintenance if we can't check
	}

	tags := machine.Tags()
	for _, tag := range tags {
		if tag == maintenance.TagHostMaintenance || tag == maintenance.TagHostNoSchedule {
			return true
		}
	}
	return false
}

// machineGetter narrows the client interface to only what is needed here
type machineGetter interface {
	Machines() maasclient.Machines
}

// lxdHostSelectorClient extends machineGetter with VMHosts for CP distribution counting
type lxdHostSelectorClient interface {
	machineGetter
	VMHosts() maasclient.VMHosts
}

// SelectOptions holds placement constraints and hints for LXD host selection.
// Semantics align with MAAS allocator params for BM: each field is optional;
// when set, it is enforced strictly; when zero/empty, that dimension is unconstrained.
type SelectOptions struct {
	Zone         string   // FailureDomain / AZ; empty = any
	ResourcePool string   // MAAS resource pool; empty = any
	Tags         []string // All required on VM host (host.Tags()); nil/empty = any
	MinCores     int      // Minimum available cores on host; 0 = no minimum
	MinMemory    int      // Minimum available memory (MB) on host; 0 = no minimum

	// ClusterID When set (non-empty), the selector counts how many control-plane VMs for this
	// cluster already exist on each eligible host. Hosts with fewer existing CP VMs
	// are preferred, distributing CP VMs across multiple physical hosts.
	ClusterID string
}

// candidateHost holds a VM host along with computed metrics for ranking.
// The cpCount field is used for anti-affinity: hosts with lower cpCount are preferred
// to spread control-plane VMs across multiple physical hosts.
type candidateHost struct {
	host    maasclient.VMHost
	cpCount int // number of CP VMs for the target cluster already on this host (0 = ideal for anti-affinity)
}

// SelectLXDHostWithMaasClient selects an LXD host based on SelectOptions.
// It implements a filter-then-rank approach:
//   - Filter: zone, pool, tags (all must match when set), resources, health, maintenance
//   - Rank: fewer CP VMs for cluster → more available memory → more available cores → managed host
//
// No fallback or constraint relaxation: if no host passes all filters, an error is returned.
func SelectLXDHostWithMaasClient(client lxdHostSelectorClient, hosts []maasclient.VMHost, opts SelectOptions) (maasclient.VMHost, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())
	ctx := context.Background()

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no LXD hosts available")
	}

	// Build cluster tag for CP distribution counting.
	// When ClusterID is set, we look for VMs tagged with both:
	//   - TagVMControlPlane ("maas-lxd-wlc-cp"): marks VM as a control-plane node
	//   - TagVMClusterPrefix + clusterID: identifies which cluster the VM belongs to
	// This allows us to count how many CP VMs from THIS cluster are already on each host.
	clusterTag := ""
	if opts.ClusterID != "" {
		clusterTag = maintenance.TagVMClusterPrefix + maintenance.SanitizeID(opts.ClusterID)
	}

	// Filter phase: collect eligible hosts
	var candidates []candidateHost
	for _, host := range hosts {
		// 1. Zone filter
		if opts.Zone != "" {
			hostZone := ""
			if host.Zone() != nil {
				hostZone = host.Zone().Name()
			}
			if hostZone != opts.Zone {
				continue
			}
		}

		// 2. Resource pool filter
		if opts.ResourcePool != "" {
			hostPool := ""
			if host.ResourcePool() != nil {
				hostPool = host.ResourcePool().Name()
			}
			if hostPool != opts.ResourcePool {
				continue
			}
		}

		// 3. Tags filter: host must have ALL specified tags
		if len(opts.Tags) > 0 {
			if !hostHasAllTags(host, opts.Tags) {
				continue
			}
		}

		// 4. Resource filter: check available cores and memory
		if opts.MinCores > 0 && host.AvailableCores() < opts.MinCores {
			log.Info("Skipping host: insufficient cores", "host", host.Name(),
				"available", host.AvailableCores(), "required", opts.MinCores)
			continue
		}
		if opts.MinMemory > 0 && host.AvailableMemory() < opts.MinMemory {
			log.Info("Skipping host: insufficient memory", "host", host.Name(),
				"available", host.AvailableMemory(), "required", opts.MinMemory)
			continue
		}

		// 5. Health check: backing machine must be powered on and Deployed
		hostSystemID := host.HostSystemID()
		if hostSystemID == "" {
			continue
		}
		machine, err := client.Machines().Machine(hostSystemID).Get(ctx)
		if err != nil {
			log.Info("Failed to get backing machine", "host", host.Name(), "system-id", hostSystemID, "error", err.Error())
			continue
		}
		if machine.PowerState() != "on" || machine.State() != "Deployed" {
			log.Info("Skipping unhealthy host", "host", host.Name(),
				"power", machine.PowerState(), "state", machine.State())
			continue
		}

		// 6. Maintenance check
		if hasMaintenanceTag(machine.Tags()) {
			log.Info("Skipping host under maintenance", "host", host.Name())
			continue
		}

		cpCount := 0
		if clusterTag != "" {
			var err error
			cpCount, err = countCPVMsOnHost(ctx, client, host, clusterTag, log)
			if err != nil {
				log.Info("Skipping host: failed to count CP VMs", "host", host.Name(), "error", err.Error())
				continue
			}
		}

		candidates = append(candidates, candidateHost{
			host:    host,
			cpCount: cpCount,
		})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no eligible LXD host found (zone=%q, pool=%q, tags=%v, minCores=%d, minMemory=%d)",
			opts.Zone, opts.ResourcePool, opts.Tags, opts.MinCores, opts.MinMemory)
	}

	// Rank phase: sort candidates by preference to select the best host.
	//
	// Priority order (highest to lowest):
	//   1. Anti-affinity: Prefer hosts with fewer CP VMs for this cluster.
	//   2. Resource availability: Prefer hosts with more available memory, then cores.
	//      Among hosts with equal CP counts, pick the one with most resource.
	//   3. Managed preference: Prefer Palette-managed hosts (lxd-host-*) over OOB hosts.
	//      Tie-breaker when all else is equal.
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]

		if a.cpCount != b.cpCount {
			return a.cpCount < b.cpCount
		}

		if a.host.AvailableMemory() != b.host.AvailableMemory() {
			return a.host.AvailableMemory() > b.host.AvailableMemory()
		}

		if a.host.AvailableCores() != b.host.AvailableCores() {
			return a.host.AvailableCores() > b.host.AvailableCores()
		}

		// 4. Prefer managed host over OOB
		return isManagedHost(a.host) && !isManagedHost(b.host)
	})

	selected := candidates[0].host
	log.Info("Selected LXD host", "host", selected.Name(), "host-id", selected.SystemID(),
		"zone", opts.Zone, "pool", opts.ResourcePool, "cpCount", candidates[0].cpCount,
		"availableCores", selected.AvailableCores(), "availableMemory", selected.AvailableMemory())
	return selected, nil
}

// hostHasAllTags checks if the VM host has all the required tags
func hostHasAllTags(host maasclient.VMHost, requiredTags []string) bool {
	hostTags := host.Tags()
	hostTagSet := make(map[string]struct{}, len(hostTags))
	for _, t := range hostTags {
		hostTagSet[t] = struct{}{}
	}
	for _, required := range requiredTags {
		if _, ok := hostTagSet[required]; !ok {
			return false
		}
	}
	return true
}

// hasMaintenanceTag checks if tags contain maintenance or no-schedule tags
func hasMaintenanceTag(tags []string) bool {
	for _, tag := range tags {
		if tag == maintenance.TagHostMaintenance || tag == maintenance.TagHostNoSchedule {
			return true
		}
	}
	return false
}

// isManagedHost checks if the host is a Palette-managed LXD host by naming convention
func isManagedHost(h maasclient.VMHost) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(h.Name())), "lxd-host-")
}

// countCPVMsOnHost counts control-plane VMs for a given cluster on the specified host.
//
//   - It queries all VMs currently running on the given LXD host
//   - It identifies CP VMs by checking for TagVMControlPlane ("maas-lxd-wlc-cp")
//   - It filters to only THIS cluster's VMs using the cluster-specific tag
//   - The returned count is used to prefer hosts with fewer existing CP VMs
//
// Example scenario with 3 LXD hosts creating a 3-node control plane:
//
//	Host A: 1 CP VM (tagged maas-lxd-wlc-cp + maas-lxd-wlc-<cluster-id>)
//	Host B: 1 CP VM (same tags)
//	Host C: 0 CP VMs
//
// When placing the 3rd CP VM, this function returns:
//
//	countCPVMsOnHost(hostA) = 1
//	countCPVMsOnHost(hostB) = 1
//	countCPVMsOnHost(hostC) = 0  ← selected (lowest count)
//
// Result: CP VMs distributed across all 3 hosts, no SPOF.
//
// Returns an error if VM listing fails, so the caller can skip this host
// rather than incorrectly treating it as having 0 CP VMs.
func countCPVMsOnHost(ctx context.Context, client lxdHostSelectorClient, host maasclient.VMHost, clusterTag string, log logr.Logger) (int, error) {
	// Get VMs on this host
	vms, err := client.VMHosts().VMHost(host.SystemID()).Machines().List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list VMs on host %s: %w", host.Name(), err)
	}

	count := 0
	for _, vm := range vms {
		// Get full VM details to access tags
		vmDetails, err := vm.Get(ctx)
		if err != nil {
			// Skip individual VM fetch errors - VM may be in transition
			log.V(1).Info("Failed to get VM details, skipping", "vm", vm.SystemID(), "error", err.Error())
			continue
		}
		tags := vmDetails.Tags()

		// Check if this VM is a control-plane node for the target cluster.
		// Both tags must be present:
		//   - TagVMControlPlane: identifies VM as a control-plane node
		//   - clusterTag: identifies which cluster this CP node belongs to
		// This ensures we only count CP VMs from the SAME cluster, not other
		// clusters that may share the same LXD host infrastructure.
		hasCP := false
		hasCluster := false
		for _, tag := range tags {
			if tag == maintenance.TagVMControlPlane {
				hasCP = true
			}
			if tag == clusterTag {
				hasCluster = true
			}
		}
		if hasCP && hasCluster {
			count++
		}
	}
	return count, nil
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
