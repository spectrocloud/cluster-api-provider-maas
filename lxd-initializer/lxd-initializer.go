package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

// Common LXD socket paths
var lxdSocketPaths = []string{
	"/var/snap/lxd/common/lxd/unix.socket", // Snap path
}

var lxdSocketPathsLegacy = []string{
	"/var/lib/lxd/unix.socket",             // Default path
	"/var/snap/lxd/common/lxd/unix.socket", // Snap installation path
	"/run/lxd.socket",                      // Alternative path
}

// auto-detection helpers
// getDefaultIface returns interface name owning default route
func getDefaultIface() (string, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) > 2 && fields[1] == "00000000" {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no default route found")
}

func linkExists(name string) bool {
	return exec.Command("ip", "link", "show", name).Run() == nil
}

// isLinuxBridge returns true if the given interface is a Linux bridge
func isLinuxBridge(name string) bool {
	_, err := os.Stat("/sys/class/net/" + name + "/bridge")
	return err == nil
}

// autoUplink returns the interface owning the default route or a safe fallback
func autoUplink() string {
	if u, err := getDefaultIface(); err == nil && u != "" {
		return u
	}
	return "enp2s0f0"
}

// ResourcePool represents a MAAS resource pool
type ResourcePool struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func main() {
	log.Println("Starting LXD initializer")

	// Define command-line flags
	action := flag.String("action", "", "Action to perform: init, register, or both")
	storageBackendFlag := flag.String("storage-backend", "", "Storage backend (dir, zfs)")
	storageSizeFlag := flag.String("storage-size", "", "Storage size in GB")
	networkBridgeFlag := flag.String("network-bridge", "", "Network bridge name")
	skipNetworkUpdateFlag := flag.Bool("skip-network-update", false, "Skip updating existing network")
	nodeIPFlag := flag.String("node-ip", "", "Node IP address for registration")
	maasEndpointFlag := flag.String("maas-endpoint", "", "MAAS API endpoint")
	maasAPIKeyFlag := flag.String("maas-api-key", "", "MAAS API key")
	zoneFlag := flag.String("zone", "", "MAAS zone for VM host")
	resourcePoolFlag := flag.String("resource-pool", "", "MAAS resource pool for VM host")
	trustPasswordFlag := flag.String("trust-password", "", "Trust password for LXD")

	flag.Parse()

	// Get environment variables if flags are not set
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Println("NODE_NAME environment variable is not set")
	}

	nodeIP := *nodeIPFlag
	if nodeIP == "" {
		nodeIP = os.Getenv("NODE_IP")
		if nodeIP == "" && nodeName != "" {
			nodeIP = nodeName // Fallback to node name if IP is not provided
		}
	}

	storageBackend := *storageBackendFlag
	if storageBackend == "" {
		storageBackend = os.Getenv("STORAGE_BACKEND")
		if storageBackend == "" {
			storageBackend = "zfs" // Default to ZFS
		}
	}

	storageSize := *storageSizeFlag
	if storageSize == "" {
		storageSize = os.Getenv("STORAGE_SIZE")
		if storageSize == "" {
			storageSize = "50" // Default to 50GB
		}
	}

	nicType := os.Getenv("NIC_TYPE")
	nicParent := os.Getenv("NIC_PARENT")

	networkBridge := *networkBridgeFlag
	if networkBridge == "" {

		networkBridge = os.Getenv("NETWORK_BRIDGE")
		if networkBridge == "" {
			networkBridge = "br0" // Default to br0
		}
	}

	// Determine final NIC behaviour
	switch nicType {
	case "bridged":
		if nicParent == "" {
			nicParent = networkBridge
		}
		if !isLinuxBridge(nicParent) {
			log.Printf("Parent %s is not a bridge – falling back to macvlan", nicParent)
			nicType, nicParent = "macvlan", autoUplink()
		}
	case "macvlan":
		if nicParent == "" {
			nicParent = autoUplink()
		}
	default: // empty or unknown -> full auto
		if networkBridge != "" && isLinuxBridge(networkBridge) {
			nicType, nicParent = "bridged", networkBridge
		} else if isLinuxBridge("br0") {
			nicType, nicParent = "bridged", "br0"
		} else {
			nicType, nicParent = "macvlan", autoUplink()
		}
	}
	log.Printf("Using NIC type=%s parent=%s", nicType, nicParent)

	skipNetworkUpdate := *skipNetworkUpdateFlag
	if !skipNetworkUpdate {
		skipNetworkUpdateEnv := os.Getenv("SKIP_NETWORK_UPDATE")
		if skipNetworkUpdateEnv == "true" || skipNetworkUpdateEnv == "1" || skipNetworkUpdateEnv == "yes" {
			skipNetworkUpdate = true
		}
	}

	maasAPIKey := *maasAPIKeyFlag
	if maasAPIKey == "" {
		maasAPIKey = os.Getenv("MAAS_API_KEY")
	}

	maasEndpoint := *maasEndpointFlag
	if maasEndpoint == "" {
		maasEndpoint = os.Getenv("MAAS_ENDPOINT")
	}

	zone := *zoneFlag
	if zone == "" {
		zone = os.Getenv("ZONE")
	}

	resourcePool := *resourcePoolFlag
	if resourcePool == "" {
		resourcePool = os.Getenv("RESOURCE_POOL")
	}

	trustPassword := *trustPasswordFlag
	if trustPassword == "" {
		trustPassword = os.Getenv("TRUST_PASSWORD")
	}

	// Determine action based on flag or default to both
	actionStr := *action
	if actionStr == "" {
		actionStr = "both" // Default to both init and register
	}

	// Perform actions based on the specified action
	if actionStr == "init" || actionStr == "both" {
		// Initialize LXD
		if err := initializeLXD(storageBackend, storageSize, networkBridge, skipNetworkUpdate, trustPassword, nicType, nicParent); err != nil {
			log.Fatalf("Failed to initialize LXD: %v", err)
		}
	}

	// if actionStr == "register" || actionStr == "both" {
	// 	// Register with MAAS if API key and endpoint are provided
	// 	if maasAPIKey != "" && maasEndpoint != "" {
	// 		if err := registerWithMAAS(nodeIP, maasAPIKey, maasEndpoint, zone, resourcePool); err != nil {
	// 			log.Fatalf("Failed to register with MAAS: %v", err)
	// 		}
	// 	} else {
	// 		log.Println("Skipping MAAS registration: MAAS_API_KEY or MAAS_ENDPOINT not provided")
	// 	}
	// }

	// If running as a standalone binary, exit after completing the actions
	if actionStr == "once" {
		log.Println("Actions completed successfully")
		return
	}

	// Keep the container running to maintain the DaemonSet if in daemon mode
	log.Println("LXD initialization completed successfully")
	log.Println("Starting periodic trust-password maintainer")
	go func() {
		for {
			if err := setTrustPassword(trustPassword); err != nil {
				log.Printf("periodic trust password set failed: %v", err)
			}
			time.Sleep(15 * time.Minute)
		}
	}()
	log.Println("Entering sleep loop to keep the container running")
	for {
		time.Sleep(3600 * time.Second)
	}
}

// findLXDSocket finds the LXD socket path
func findLXDSocket() (string, error) {
	for _, socketPath := range lxdSocketPaths {
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath, nil
		}
	}
	return "", fmt.Errorf("no LXD socket found at any of the expected paths")
}

// waitForLXDSocket keeps checking well-known paths until a socket is found or the timeout expires.
func waitForLXDSocket(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p, err := findLXDSocket(); err == nil {
			return p, nil
		}
		time.Sleep(time.Second)
	}
	return "", fmt.Errorf("timed out waiting (%s) for LXD socket", timeout)
}

// logLXDDiagnostics prints useful information when the LXD socket cannot be found.
func logLXDDiagnostics() {
	log.Println("===== LXD diagnostics =====")
	for _, p := range lxdSocketPaths {
		if fi, err := os.Stat(p); err == nil {
			log.Printf("candidate %s exists (size %d)", p, fi.Size())
		} else if os.IsNotExist(err) {
			log.Printf("candidate %s missing", p)
		} else {
			log.Printf("candidate %s error: %v", p, err)
		}
	}

	// host daemon status
	out, err := exec.Command("nsenter", "-t", "1", "-m", "-p", "--", "systemctl", "status", "snap.lxd.daemon").CombinedOutput()
	if err == nil {
		log.Printf("systemctl status snap.lxd.daemon:\n%s", string(out))
	} else {
		log.Printf("nsenter systemctl status failed: %v", err)
	}

	// process list
	if psOut, err := exec.Command("ps", "-ef").CombinedOutput(); err == nil {
		log.Printf("process list:\n%s", string(psOut))
	}
	log.Println("===== end diagnostics =====")
}

// setTrustPassword sets core.trust_password and verifies it.
func setTrustPassword(pw string) error {
	if pw == "" {
		return nil
	}
	// try inside container first
	cmd := exec.Command("/snap/bin/lxc", "config", "set", "core.trust_password", pw)
	if _, err := cmd.CombinedOutput(); err != nil {
		log.Printf("lxc trust_password failed inside container (%v), trying nsenter", err)
		cmd = exec.Command("nsenter", "-t", "1", "-m", "-p", "--", "/snap/bin/lxc", "config", "set", "core.trust_password", pw)
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to set trust password: %s: %w", string(out2), err2)
		}
	}
	// verify
	verifyCmd := exec.Command("nsenter", "-t", "1", "-m", "-p", "--", "/snap/bin/lxc", "config", "get", "core.trust_password")
	verifyOut, err := verifyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("verification get failed: %s: %w", string(verifyOut), err)
	}
	if strings.TrimSpace(string(verifyOut)) != "true" {
		return fmt.Errorf("verification failed, expected 'true', got %q", strings.TrimSpace(string(verifyOut)))
	}
	log.Println("trust password verified present")
	return nil
}

// initializeLXD initializes LXD with the specified configuration
func initializeLXD(storageBackend, storageSize, networkBridge string, skipNetworkUpdate bool, trustPassword, nicType, nicParent string) error {
	log.Printf("Initializing LXD with storage backend %s, size %sGB, and network bridge %s",
		storageBackend, storageSize, networkBridge)

	if skipNetworkUpdate {
		log.Println("Network updates will be skipped for existing networks")
	}

	// Wait for LXD socket to appear (up to 2 minutes)
	socketPath, err := waitForLXDSocket(2 * time.Minute)
	if err != nil {
		logLXDDiagnostics()
		return err
	}

	log.Printf("Found LXD socket at %s", socketPath)

	// Connect to LXD
	c, err := lxdclient.ConnectLXDUnix(socketPath, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to LXD: %w", err)
	}

	// Check if LXD is already initialized
	server, _, err := c.GetServer()
	if err != nil {
		return fmt.Errorf("failed to get server info: %w", err)
	}

	// Check LXD version to determine the approach
	apiVersion := server.APIVersion
	log.Printf("LXD API Version: %s", apiVersion)

	// Check for storage pools and networks
	storagePoolsResponse, err := c.GetStoragePools()
	if err != nil {
		log.Printf("Warning: Failed to get storage pools: %v", err)
	}

	networksResponse, err := c.GetNetworks()
	if err != nil {
		log.Printf("Warning: Failed to get networks: %v", err)
	}

	// Check if storage pool exists
	storagePoolExists := false
	storagePoolName := "default"

	for _, pool := range storagePoolsResponse {
		if pool.Name == storagePoolName {
			storagePoolExists = true
			log.Printf("Storage pool '%s' already exists with driver '%s'", pool.Name, pool.Driver)
			break
		}
	}

	if !storagePoolExists {
		log.Printf("Creating storage pool '%s' with driver '%s'", storagePoolName, storageBackend)

		// Create storage pool
		storagePoolReq := api.StoragePoolsPost{
			Name:   storagePoolName,
			Driver: storageBackend,
			StoragePoolPut: api.StoragePoolPut{
				Config: map[string]string{
					"size": storageSize + "GB",
				},
			},
		}
		if err = c.CreateStoragePool(storagePoolReq); err != nil {
			return fmt.Errorf("failed to create storage pool: %w", err)
		}
		log.Printf("Storage pool '%s' created successfully", storagePoolName)
	} else {
		// pool exists, ensure size matches requested
		pool, etag, err := c.GetStoragePool(storagePoolName)
		if err == nil {
			desired := storageSize + "GB"
			if pool.Config["size"] != desired {
				log.Printf("Updating storage pool '%s' size from %s to %s", storagePoolName, pool.Config["size"], desired)
				poolPut := pool.Writable()
				if poolPut.Config == nil {
					poolPut.Config = map[string]string{}
				}
				poolPut.Config["size"] = desired
				if err := c.UpdateStoragePool(storagePoolName, poolPut, etag); err != nil {
					log.Printf("Warning: failed to update storage pool size: %v", err)
				} else {
					log.Printf("Storage pool '%s' size updated", storagePoolName)
				}
			}
		}
	}

	// Network configuration based on nicType
	if nicType == "bridge" {
		bridgeName := nicParent
		// Check if network bridge exists
		networkExists := false
		var existingNetworkType string

		for _, network := range networksResponse {
			if network.Name == bridgeName {
				networkExists = true
				existingNetworkType = network.Type
				log.Printf("Network '%s' already exists with type '%s'", network.Name, network.Type)
				break
			}
		}

		if !networkExists {
			log.Printf("Creating network '%s'", bridgeName)

			// Create network
			networkReq := api.NetworksPost{
				Name: bridgeName,
				Type: "bridge",
				NetworkPut: api.NetworkPut{
					Config: map[string]string{
						"ipv4.address": "auto",
						"ipv4.nat":     "true", // Using string instead of boolean
					},
				},
			}

			err = c.CreateNetwork(networkReq)
			if err != nil {
				return fmt.Errorf("failed to create network: %w", err)
			}
			log.Printf("Network '%s' created successfully", bridgeName)
		} else if !skipNetworkUpdate {
			// Only try to update if it's a managed bridge
			if existingNetworkType == "bridge" {
				// Update existing network
				network, etag, err := c.GetNetwork(bridgeName)
				if err != nil {
					log.Printf("Warning: Failed to get network details for update: %v", err)
					log.Println("Skipping network update")
				} else {
					// Ensure ipv4.nat is set to "true" as a string
					networkPut := api.NetworkPut{
						Config: network.Config,
					}

					// Only set if not already set
					if networkPut.Config == nil {
						networkPut.Config = make(map[string]string)
					}

					networkPut.Config["ipv4.nat"] = "true" // Using string instead of boolean

					err = c.UpdateNetwork(bridgeName, networkPut, etag)
					if err != nil {
						log.Printf("Warning: Failed to update network: %v", err)
						log.Println("Continuing without network update")
					} else {
						log.Printf("Network '%s' updated successfully", bridgeName)
					}
				}
			} else {
				log.Printf("Network '%s' is type '%s', not updating (only bridge networks can be updated)",
					bridgeName, existingNetworkType)
			}
		} else {
			log.Printf("Network '%s' exists and skip-network-update is set, skipping update", bridgeName)
		}

	} // end if nicType == "bridge"

	// Ensure MAAS VM profile exists with disk+nic
	if err := ensureMAASProfile(c, nicType, nicParent, storagePoolName); err != nil {
		log.Printf("Warning: Failed to ensure MAAS profile: %v", err)
	}

	// Configure LXD to listen on the network
	if err := configureLXDNetwork(trustPassword); err != nil {
		log.Printf("Warning: Failed to configure LXD network: %v", err)
		log.Println("Continuing anyway, as this is not critical")
	}

	return nil
}

// configureLXDNetwork configures LXD to listen on the network
// ensureMAASProfile makes sure a profile named "maas-kvm" exists with root disk and bridged nic
func ensureMAASProfile(c lxdclient.InstanceServer, nicType, nicParent, pool string) error {
	const profileName = "maas-kvm"
	profiles, err := c.GetProfiles()
	if err != nil {
		return fmt.Errorf("list profiles: %w", err)
	}
	for _, p := range profiles {
		if p.Name == profileName {
			return nil // already present
		}
	}
	profile := api.ProfilesPost{
		Name: profileName,
		ProfilePut: api.ProfilePut{
			Description: "Profile used by MAAS for KVM VMs",
			Devices: map[string]map[string]string{
				"root": {
					"type": "disk",
					"pool": pool,
					"path": "/",
				},
				"eth0": {
					"type":    "nic",
					"nictype": nicType,
					"parent":  nicParent,
					"name":    "eth0",
				},
			},
		},
	}
	if err := c.CreateProfile(profile); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

func configureLXDNetwork(trustPassword string) error {
	log.Println("Configuring LXD to listen on the network")

	// Set LXD to listen on all interfaces on port 8443
	cmd := exec.Command("/snap/bin/lxc", "config", "set", "core.https_address", ":8443")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("lxc command failed (%v), trying nsenter fallback", err)
		cmd = exec.Command("nsenter", "-t", "1", "-m", "-p", "--", "/snap/bin/lxc", "config", "set", "core.https_address", ":8443")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to set core.https_address (fallback): %s: %w", string(output), err)
		}
	}

	// Also set trust password (verification inside helper)
	if trustPassword != "" {
		if err := setTrustPassword(trustPassword); err != nil {
			return err
		}
	}
	// Restart LXD to apply changes
	cmd = exec.Command("systemctl", "restart", "snap.lxd.daemon")
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("systemctl restart inside container failed (%v), trying nsenter fallback", err)
		cmd = exec.Command("nsenter", "-t", "1", "-m", "-p", "--", "systemctl", "restart", "snap.lxd.daemon")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to restart LXD (fallback): %s: %w", string(output), err)
		}
	}

	log.Println("LXD configured to listen on port 8443")
	return nil
}

// extractLXDCertificateAndKey extracts the LXD certificate and key to the specified paths
func extractLXDCertificateAndKey(certPath, keyPath string) error {
	// Copy the certificate
	cmd := exec.Command("cp", "/var/snap/lxd/common/lxd/server.crt", certPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative path
		cmd = exec.Command("cp", "/var/lib/lxd/server.crt", certPath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to copy certificate: %s: %w", string(output), err)
		}
	}

	// Copy the key
	cmd = exec.Command("cp", "/var/snap/lxd/common/lxd/server.key", keyPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Try alternative path
		cmd = exec.Command("cp", "/var/lib/lxd/server.key", keyPath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to copy key: %s: %w", string(output), err)
		}
	}

	// Fix permissions
	cmd = exec.Command("chmod", "644", certPath, keyPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set permissions: %s: %w", string(output), err)
	}

	return nil
}

// getResourcePools gets the available resource pools from MAAS
func getResourcePools(maasEndpoint, maasAPIKey string) ([]ResourcePool, error) {
	log.Println("Getting available resource pools from MAAS")

	// Get MAAS profile name from API key
	profileName := ""
	if maasAPIKey != "" {
		parts := strings.Split(maasAPIKey, ":")
		if len(parts) > 0 {
			profileName = parts[0]
		}
	}

	if profileName == "" {
		profileName = "admin" // Default profile name
	}

	// Use maas CLI to get resource pools
	cmd := exec.Command("maas", profileName, "resource-pools", "read")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If CLI fails, try direct HTTP request
		return getResourcePoolsViaHTTP(maasEndpoint, maasAPIKey)
	}

	// Parse JSON output
	var pools []ResourcePool
	err = json.Unmarshal(output, &pools)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource pools: %w", err)
	}

	// Print available pools
	log.Println("Available resource pools:")
	for _, pool := range pools {
		log.Printf("  - %s (ID: %d)", pool.Name, pool.ID)
	}

	return pools, nil
}

// getResourcePoolsViaHTTP gets the available resource pools from MAAS via HTTP
func getResourcePoolsViaHTTP(maasEndpoint, maasAPIKey string) ([]ResourcePool, error) {
	// Construct the URL
	endpoint := fmt.Sprintf("%s/api/2.0/resource-pools/", strings.TrimSuffix(maasEndpoint, "/"))

	// Create the request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add OAuth header
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", maasAPIKey))

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("MAAS API returned non-OK status: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var pools []ResourcePool
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Print available pools
	log.Println("Available resource pools:")
	for _, pool := range pools {
		log.Printf("  - %s (ID: %d)", pool.Name, pool.ID)
	}

	return pools, nil
}

// // registerWithMAAS registers the host with MAAS as a VM host
// func registerWithMAAS(nodeIP, maasAPIKey, maasEndpoint, zone, resourcePool string) error {
// 	log.Printf("Registering host %s with MAAS", nodeIP)

// 	if nodeIP == "" {
// 		return fmt.Errorf("node IP is required for MAAS registration")
// 	}

// 	// Extract certificate and key
// 	certPath := "/tmp/lxd.crt"
// 	keyPath := "/tmp/lxd.key"

// 	err := extractLXDCertificateAndKey(certPath, keyPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to extract LXD certificate and key: %w", err)
// 	}

// 	// Auto-detect resource pool if not specified
// 	if resourcePool == "" {
// 		pools, err := getResourcePools(maasEndpoint, maasAPIKey)
// 		if err != nil {
// 			log.Printf("Warning: Failed to get resource pools: %v", err)
// 			log.Println("Using default resource pool")
// 			resourcePool = "default"
// 		} else if len(pools) > 0 {
// 			resourcePool = pools[0].Name
// 			log.Printf("Auto-detected resource pool: %s", resourcePool)
// 		} else {
// 			log.Println("No resource pools found, using default")
// 			resourcePool = "default"
// 		}
// 	}

// 	// Get MAAS profile name from API key
// 	profileName := ""
// 	if maasAPIKey != "" {
// 		parts := strings.Split(maasAPIKey, ":")
// 		if len(parts) > 0 {
// 			profileName = parts[0]
// 		}
// 	}

// 	if profileName == "" {
// 		profileName = "admin" // Default profile name
// 	}

// 	// Login to MAAS
// 	cmd := exec.Command("maas", "login", profileName, maasEndpoint, maasAPIKey)
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return fmt.Errorf("failed to login to MAAS: %s: %w", string(output), err)
// 	}

// 	// Register the host with MAAS
// 	powerAddress := fmt.Sprintf("https://%s:8443", nodeIP)
// 	hostName := fmt.Sprintf("lxd-host-%s", nodeIP)

// 	args := []string{
// 		profileName, "vm-hosts", "create",
// 		"type=lxd",
// 		"power_address=" + powerAddress,
// 		"power_user=root",
// 		"name=" + hostName,
// 		"pool=" + resourcePool,
// 		"project=default",
// 		"tags=lxd-host,capmaas",
// 	}

// 	// Add zone if specified
// 	if zone != "" {
// 		args = append(args, "zone="+zone)
// 	}

// 	// Add certificate and key
// 	args = append(args, "certificate=@"+certPath)
// 	args = append(args, "key=@"+keyPath)

// 	cmd = exec.Command("maas", args...)
// 	output, err = cmd.CombinedOutput()
// 	if err != nil {
// 		return fmt.Errorf("failed to register VM host: %s: %w", string(output), err)
// 	}

// 	log.Printf("LXD host registered successfully with MAAS: %s", hostName)
// 	return nil
// }
