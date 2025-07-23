package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

// Common LXD socket paths
var lxdSocketPaths = []string{
	"/var/lib/lxd/unix.socket",             // Default path
	"/var/snap/lxd/common/lxd/unix.socket", // Snap installation path
	"/run/lxd.socket",                      // Alternative path
}

func main() {
	log.Println("Starting LXD initializer")

	// Get environment variables
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable is required")
	}

	storageBackend := os.Getenv("STORAGE_BACKEND")
	if storageBackend == "" {
		storageBackend = "zfs" // Default to ZFS
	}

	storageSize := os.Getenv("STORAGE_SIZE")
	if storageSize == "" {
		storageSize = "50" // Default to 50GB
	}

	networkBridge := os.Getenv("NETWORK_BRIDGE")
	if networkBridge == "" {
		networkBridge = "br0" // Default to br0
	}

	maasAPIKey := os.Getenv("MAAS_API_KEY")
	maasEndpoint := os.Getenv("MAAS_ENDPOINT")
	zone := os.Getenv("ZONE")
	resourcePool := os.Getenv("RESOURCE_POOL")

	// Initialize LXD
	if err := initializeLXD(storageBackend, storageSize, networkBridge); err != nil {
		log.Fatalf("Failed to initialize LXD: %v", err)
	}

	// Register with MAAS if API key and endpoint are provided
	if maasAPIKey != "" && maasEndpoint != "" {
		if err := registerWithMAAS(nodeName, maasAPIKey, maasEndpoint, zone, resourcePool); err != nil {
			log.Fatalf("Failed to register with MAAS: %v", err)
		}
	}

	// Keep the container running to maintain the DaemonSet
	log.Println("LXD initialization completed successfully")
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

// initializeLXD initializes LXD with the specified configuration
func initializeLXD(storageBackend, storageSize, networkBridge string) error {
	log.Printf("Initializing LXD with storage backend %s, size %sGB, and network bridge %s",
		storageBackend, storageSize, networkBridge)

	// Find LXD socket
	socketPath, err := findLXDSocket()
	if err != nil {
		return fmt.Errorf("failed to find LXD socket: %w", err)
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

		err = c.CreateStoragePool(storagePoolReq)
		if err != nil {
			return fmt.Errorf("failed to create storage pool: %w", err)
		}
		log.Printf("Storage pool '%s' created successfully", storagePoolName)
	}

	// Check if network bridge exists
	networkExists := false

	for _, network := range networksResponse {
		if network.Name == networkBridge {
			networkExists = true
			log.Printf("Network '%s' already exists", network.Name)
			break
		}
	}

	if !networkExists {
		log.Printf("Creating network '%s'", networkBridge)

		// Create network
		networkReq := api.NetworksPost{
			Name: networkBridge,
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
		log.Printf("Network '%s' created successfully", networkBridge)
	} else {
		// Update existing network
		network, etag, err := c.GetNetwork(networkBridge)
		if err != nil {
			return fmt.Errorf("failed to get network details: %w", err)
		}

		// Ensure ipv4.nat is set to "true" as a string
		networkPut := api.NetworkPut{
			Config: network.Config,
		}
		networkPut.Config["ipv4.nat"] = "true" // Using string instead of boolean

		err = c.UpdateNetwork(networkBridge, networkPut, etag)
		if err != nil {
			return fmt.Errorf("failed to update network: %w", err)
		}
		log.Printf("Network '%s' updated successfully", networkBridge)
	}

	return nil
}

// registerWithMAAS registers the host with MAAS as a VM host
func registerWithMAAS(nodeName, maasAPIKey, maasEndpoint, zone, resourcePool string) error {
	log.Printf("Registering host %s with MAAS", nodeName)

	// Get the node's IP address
	// In a real implementation, you'd get this from the node's network interface
	// For now, we'll use the node name as a placeholder
	nodeIP := nodeName

	// Check if host is already registered
	// This is a simplified implementation - in a real scenario, you'd make an HTTP request to the MAAS API
	log.Printf("Checking if host %s is already registered with MAAS", nodeIP)

	// Register the host with MAAS
	log.Printf("Registering host %s with MAAS endpoint %s", nodeIP, maasEndpoint)

	// Create registration parameters
	params := url.Values{}
	params.Add("type", "lxd")
	params.Add("power_address", fmt.Sprintf("https://%s:8443", nodeIP))
	params.Add("name", fmt.Sprintf("lxd-host-%s", nodeIP))

	if zone != "" {
		params.Add("zone", zone)
	}

	if resourcePool != "" {
		params.Add("pool", resourcePool)
	}

	// In a real implementation, you'd make an HTTP request to the MAAS API
	// For now, we'll just log what we would do
	log.Printf("Would make POST request to %s/MAAS/api/2.0/vm-hosts/", maasEndpoint)
	log.Printf("With Authorization header: ApiKey %s", maasAPIKey)
	log.Printf("With form data:")
	for key, values := range params {
		for _, value := range values {
			log.Printf("  %s: %s", key, value)
		}
	}

	return nil
}
