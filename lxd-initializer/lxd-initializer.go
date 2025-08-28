package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/spectrocloud/maas-client-go/maasclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Common LXD socket paths
var lxdSocketPaths = []string{
	"/var/snap/lxd/common/lxd/unix.socket", // Snap path
}

// getKubernetesClient returns a Kubernetes client using in-cluster config
func getKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return clientset, nil
}

// getMaasCredentialsFromSecret reads MAAS credentials from the capmaas-manager-bootstrap-credentials secret
// It searches across all namespaces to find the secret
func getMaasCredentialsFromSecret() (string, string, error) {
	client, err := getKubernetesClient()
	if err != nil {
		return "", "", fmt.Errorf("failed to get kubernetes client: %v", err)
	}

	secretName := "capmaas-manager-bootstrap-credentials"

	// List all secrets across all namespaces
	secretList, err := client.CoreV1().Secrets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to list secrets: %v", err)
	}

	// Search for the capmaas-manager-bootstrap-credentials secret
	for _, secret := range secretList.Items {
		if secret.Name == secretName {
			maasEndpoint := string(secret.Data["MAAS_ENDPOINT"])
			maasAPIKey := string(secret.Data["MAAS_API_KEY"])

			if maasEndpoint == "" || maasAPIKey == "" {
				log.Printf("Warning: Found secret %s in namespace %s but MAAS_ENDPOINT or MAAS_API_KEY is empty", secretName, secret.Namespace)
				continue
			}

			log.Printf("Found MAAS credentials in secret %s in namespace %s", secretName, secret.Namespace)
			return maasEndpoint, maasAPIKey, nil
		}
	}

	return "", "", fmt.Errorf("secret %s not found in any namespace", secretName)
}

// extractSystemIDFromNodeName extracts system ID from MAAS node name
func extractSystemIDFromNodeName(nodeName string) (string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx := context.Background()
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	providerID := node.Spec.ProviderID
	if err != nil {
		return "", fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}
	if !strings.HasPrefix(providerID, "maas:///") {
		return "", fmt.Errorf("invalid MAAS providerID format: %s", providerID)
	}

	parts := strings.Split(providerID, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid MAAS providerID format")
	}

	systemID := parts[len(parts)-1]
	if systemID == "" {
		return "", fmt.Errorf("empty system ID in providerID: %s", providerID)
	}

	return systemID, nil
}

// getStorageSizeFromMaas retrieves storage size from MAAS for the current node
func getStorageSizeFromMaas(nodeName, maasAPIKey, maasEndpoint string) (string, error) {
	if nodeName == "" || maasAPIKey == "" || maasEndpoint == "" {
		return "", fmt.Errorf("missing required parameters")
	}
	ctx := context.Background()
	systemID, err := extractSystemIDFromNodeName(nodeName)
	if err != nil {
		return "", fmt.Errorf("failed to extract system ID: %w", err)
	}

	client := maasclient.NewAuthenticatedClientSet(maasEndpoint, maasAPIKey)
	machineClient := client.Machines().Machine(systemID)
	machine, err := machineClient.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get machine details from MAAS: %w", err)
	}

	// Get total storage size
	totalStorageGB := machine.TotalStorageGB()
	if totalStorageGB == 0 {
		return "", fmt.Errorf("no storage found")
	}

	// Use 80% of total storage for LXD pool (leave space for OS)
	lxdStorageGB := totalStorageGB * 0.8
	log.Printf("Using storage size: %.0fGB (%.0f%% of total)", lxdStorageGB, lxdStorageGB/totalStorageGB*100)

	return strconv.FormatFloat(lxdStorageGB, 'f', 0, 64), nil
}

// ResourcePool represents a MAAS resource pool
type ResourcePool struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// getMachineInfoFromMaas gets zone, resource pool, and boot interface from MAAS for the current node
func getMachineInfoFromMaas(nodeName, maasAPIKey, maasEndpoint string) (zone, resourcePool, bootInterface string, err error) {
	if nodeName == "" || maasAPIKey == "" || maasEndpoint == "" {
		return "", "", "", fmt.Errorf("missing required parameters")
	}
	ctx := context.Background()
	systemID, err := extractSystemIDFromNodeName(nodeName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to extract system ID: %w", err)
	}

	client := maasclient.NewAuthenticatedClientSet(maasEndpoint, maasAPIKey)
	machine, err := client.Machines().Machine(systemID).Get(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get machine details from MAAS: %w", err)
	}

	// Get zone
	if machine.Zone() != nil {
		zone = machine.Zone().Name()
	}

	resourcePool = machine.ResourcePoolName()
	bootInterface = machine.BootInterfaceName()

	return zone, resourcePool, bootInterface, nil
}

func main() {
	log.Println("Starting LXD initializer")

	// Define command-line flags
	action := flag.String("action", "", "Action to perform: init, register, or both")
	storageBackendFlag := flag.String("storage-backend", "", "Storage backend (dir, zfs)")
	networkBridgeFlag := flag.String("network-bridge", "", "Network bridge name")
	skipNetworkUpdateFlag := flag.Bool("skip-network-update", false, "Skip updating existing network")
	nodeIPFlag := flag.String("node-ip", "", "Node IP address for registration")
	maasEndpointFlag := flag.String("maas-endpoint", "", "MAAS API endpoint")
	maasAPIKeyFlag := flag.String("maas-api-key", "", "MAAS API key")
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

	var storageSize string

	// Auto-detect Storage Size of a bare metal machine
	maasAPIKey := *maasAPIKeyFlag
	maasEndpoint := *maasEndpointFlag

	// If flags are not provided, try to read from the Kubernetes secret
	if maasAPIKey == "" || maasEndpoint == "" {
		if secretEndpoint, secretAPIKey, err := getMaasCredentialsFromSecret(); err == nil {
			if maasEndpoint == "" {
				maasEndpoint = secretEndpoint
			}
			if maasAPIKey == "" {
				maasAPIKey = secretAPIKey
			}
		} else {
			log.Printf("Warning: Failed to get MAAS credentials from secret: %v", err)
		}
	}

	if maasAPIKey != "" && maasEndpoint != "" {
		if maasStorageSize, err := getStorageSizeFromMaas(nodeName, maasAPIKey, maasEndpoint); err == nil {
			storageSize = maasStorageSize
			log.Printf("Using storage size from MAAS: %s GB", storageSize)
		} else {
			log.Printf("Warning: Failed to get storage size from MAAS: %v, using default", err)
			storageSize = "50"
		}
	} else {
		log.Printf("Warning: MAAS API credentials not available, using default storage size")
		storageSize = "50"
	}

	nicType := "macvlan"

	networkBridge := *networkBridgeFlag
	if networkBridge == "" {
		networkBridge = os.Getenv("NETWORK_BRIDGE")
		if networkBridge == "" {
			networkBridge = "br0" // Default to br0
		}
	}

	// Auto-detect zone, resource pool, and boot interface name from MAAS
	zone, resourcePool, bootInterfaceName, err := getMachineInfoFromMaas(nodeName, maasAPIKey, maasEndpoint)
	if err != nil {
		log.Fatalf("Failed to get machine information from MAAS: %v", err)
	}
	log.Printf("Zone retrieved from MAAS: %s", zone)
	log.Printf("Resource pool retrieved from MAAS: %s", resourcePool)
	log.Printf("Boot interface retrieved from MAAS: %s", bootInterfaceName)

	nicParent := bootInterfaceName

	log.Printf("Using NIC type=%s parent=%s", nicType, nicParent)

	skipNetworkUpdate := *skipNetworkUpdateFlag
	if !skipNetworkUpdate {
		skipNetworkUpdateEnv := os.Getenv("SKIP_NETWORK_UPDATE")
		if skipNetworkUpdateEnv == "true" || skipNetworkUpdateEnv == "1" || skipNetworkUpdateEnv == "yes" {
			skipNetworkUpdate = true
		}
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

		// Mark the node as LXD initialized (production: only log errors)
		if nodeName != "" {
			nodeLabeler, err := NewNodeLabeler(nodeName)
			if err != nil {
				log.Printf("Warning: Failed to create node labeler for %s: %v", nodeName, err)
			} else {
				nodeLabeler.SafeMarkLXDInitialized()
			}
		}
	}

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
