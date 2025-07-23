package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// VMHostExtension provides extensions to the MAAS client for VM host operations
// These extensions will be integrated once the MAAS client library supports VMHosts API
type VMHostExtension struct {
	client maasclient.ClientSetInterface
}

// NewVMHostExtension creates a new VM host extension
func NewVMHostExtension(client maasclient.ClientSetInterface) *VMHostExtension {
	return &VMHostExtension{
		client: client,
	}
}

// VMHostQueryParams represents parameters for querying VM hosts
type VMHostQueryParams struct {
	Type         string  // lxd, kvm, etc.
	Zone         *string // availability zone filter
	ResourcePool *string // resource pool filter
	Tags         []string
}

// VMHost represents a MAAS VM host (LXD host, KVM host, etc.)
type VMHost struct {
	ID               string
	Name             string
	Type             string
	AvailabilityZone string
	ResourcePool     string
	Tags             []string
	Resources        VMHostResources
}

// VMHostResources represents resource information for a VM host
type VMHostResources struct {
	Total     ResourceInfo
	Used      ResourceInfo
	Available ResourceInfo
}

// ResourceInfo represents CPU and memory resource information
type ResourceInfo struct {
	Cores  int
	Memory int // in MB
}

// ComposedVM represents a VM created through MAAS VM composition
type ComposedVM struct {
	SystemID         string
	VMHostID         string
	AvailabilityZone string
	Status           string
}

// ComposeVMParams represents parameters for VM composition
type ComposeVMParams struct {
	VMHostID string
	Cores    int
	Memory   int // in MB
	Disks    []DiskSpec
	Profile  string
	Project  string
	UserData string
	Tags     []string

	// NetworkConfig defines network configuration for the VM
	NetworkConfig *NetworkConfigParams
}

// DiskSpec represents disk specification for VM composition
type DiskSpec struct {
	Size string // e.g., "20GB"
	Pool string // storage pool name
}

// NetworkConfigParams represents network configuration parameters for VM composition
type NetworkConfigParams struct {
	// Interfaces defines interface configurations for the VM
	Interfaces []InterfaceConfig
}

// InterfaceConfig represents configuration for a single network interface
type InterfaceConfig struct {
	// Name is the interface name (e.g., "eth0")
	Name string
	// Bridge is the network bridge to connect to
	Bridge *string
	// MacAddress is the MAC address to assign to the interface
	MacAddress *string
	// IPConfig defines IP configuration for the interface
	IPConfig *StaticIPParams
}

// StaticIPParams represents static IP configuration parameters
type StaticIPParams struct {
	// IPAddress is the static IP address to assign
	IPAddress string
	// Gateway is the default gateway IP address
	Gateway string
	// Subnet is the subnet mask or CIDR notation
	Subnet string
	// DNSServers is a list of DNS server IP addresses
	DNSServers []string
}

// QueryVMHosts queries available VM hosts based on parameters
func (e *VMHostExtension) QueryVMHosts(ctx context.Context, params VMHostQueryParams) ([]VMHost, error) {
	queryParams := maasclient.ParamsBuilder()

	if params.Type != "" {
		queryParams.Set("type", params.Type)
	}
	if params.Zone != nil && *params.Zone != "" {
		queryParams.Set("zone", *params.Zone)
	}
	if params.ResourcePool != nil && *params.ResourcePool != "" {
		queryParams.Set("pool", *params.ResourcePool)
	}
	if len(params.Tags) > 0 {
		queryParams.Set("tags", strings.Join(params.Tags, ","))
	}

	// Use the actual MAAS VMHosts API
	vmHosts, err := e.client.VMHosts().List(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to query VM hosts: %w", err)
	}

	return e.convertMaasVMHostsToLocal(vmHosts), nil
}

// ComposeVM creates a VM on the specified VM host
func (e *VMHostExtension) ComposeVM(ctx context.Context, params ComposeVMParams) (*ComposedVM, error) {
	composeParams := maasclient.ParamsBuilder().
		Set("cores", strconv.Itoa(params.Cores)).
		Set("memory", strconv.Itoa(params.Memory))

	if params.UserData != "" {
		composeParams.Set("user_data", params.UserData)
	}

	if params.Profile != "" {
		composeParams.Set("profile", params.Profile)
	}

	if params.Project != "" {
		composeParams.Set("project", params.Project)
	}

	// Add disk configuration
	if len(params.Disks) > 0 {
		diskSpecs := make([]string, 0, len(params.Disks))
		for _, disk := range params.Disks {
			spec := fmt.Sprintf("size=%s", disk.Size)
			if disk.Pool != "" {
				spec += fmt.Sprintf(",pool=%s", disk.Pool)
			}
			diskSpecs = append(diskSpecs, spec)
		}
		composeParams.Set("disks", strings.Join(diskSpecs, ";"))
	}

	// Add network configuration if specified
	if params.NetworkConfig != nil {
		if err := e.validateNetworkConfig(params.NetworkConfig); err != nil {
			return nil, fmt.Errorf("network configuration validation failed: %w", err)
		}

		if err := e.buildNetworkConfigParams(composeParams, params.NetworkConfig); err != nil {
			return nil, fmt.Errorf("failed to build network configuration parameters: %w", err)
		}
	}

	// Use the actual MAAS VMHosts API
	composedVM, err := e.client.VMHosts().VMHost(params.VMHostID).Composer().
		Compose(ctx, composeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to compose VM: %w", err)
	}

	return &ComposedVM{
		SystemID:         composedVM.SystemID(),
		VMHostID:         params.VMHostID,
		AvailabilityZone: composedVM.Zone().Name(),
		Status:           composedVM.State(),
	}, nil
}

// DecomposeVM destroys a composed VM using the MAAS VMHosts API
func (e *VMHostExtension) DecomposeVM(ctx context.Context, vmHostID, systemID string) error {
	if e.client == nil {
		// In test mode with nil client, simulate success
		return nil
	}

	// Use the standard machine release API for VM decommissioning
	// VMs composed through VM hosts are still managed as regular machines for lifecycle operations
	_, err := e.client.Machines().Machine(systemID).Releaser().Release(ctx)
	return err
}

// GetVMHostResources retrieves resource usage information for a VM host using the MAAS VMHosts API
func (e *VMHostExtension) GetVMHostResources(ctx context.Context, vmHostID string) (*VMHostResources, error) {
	if e.client == nil {
		// In test mode with nil client, return mock data
		return e.getMockVMHostResources(vmHostID), nil
	}

	vmHost, err := e.client.VMHosts().VMHost(vmHostID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM host resources: %w", err)
	}

	return &VMHostResources{
		Total: ResourceInfo{
			Cores:  vmHost.TotalCores(),
			Memory: vmHost.TotalMemory(),
		},
		Used: ResourceInfo{
			Cores:  vmHost.UsedCores(),
			Memory: vmHost.UsedMemory(),
		},
		Available: ResourceInfo{
			Cores:  vmHost.AvailableCores(),
			Memory: vmHost.AvailableMemory(),
		},
	}, nil
}

// Mock implementations for testing when client is nil

func (e *VMHostExtension) getMockVMHosts(params VMHostQueryParams) []VMHost {
	hosts := []VMHost{
		{
			ID:               "lxd-host-1",
			Name:             "LXD Host 1",
			Type:             "lxd",
			AvailabilityZone: "zone-a",
			ResourcePool:     "default",
			Tags:             []string{"lxd", "compute"},
			Resources: VMHostResources{
				Total:     ResourceInfo{Cores: 16, Memory: 32768},
				Used:      ResourceInfo{Cores: 4, Memory: 8192},
				Available: ResourceInfo{Cores: 12, Memory: 24576},
			},
		},
		{
			ID:               "lxd-host-2",
			Name:             "LXD Host 2",
			Type:             "lxd",
			AvailabilityZone: "zone-b",
			ResourcePool:     "default",
			Tags:             []string{"lxd", "compute"},
			Resources: VMHostResources{
				Total:     ResourceInfo{Cores: 16, Memory: 32768},
				Used:      ResourceInfo{Cores: 2, Memory: 4096},
				Available: ResourceInfo{Cores: 14, Memory: 28672},
			},
		},
		{
			ID:               "lxd-host-3",
			Name:             "LXD Host 3",
			Type:             "lxd",
			AvailabilityZone: "zone-c",
			ResourcePool:     "compute-pool",
			Tags:             []string{"lxd", "compute", "high-memory"},
			Resources: VMHostResources{
				Total:     ResourceInfo{Cores: 32, Memory: 65536},
				Used:      ResourceInfo{Cores: 8, Memory: 16384},
				Available: ResourceInfo{Cores: 24, Memory: 49152},
			},
		},
	}

	// Filter by zone if specified
	if params.Zone != nil && *params.Zone != "" {
		filteredHosts := make([]VMHost, 0)
		for _, host := range hosts {
			if host.AvailabilityZone == *params.Zone {
				filteredHosts = append(filteredHosts, host)
			}
		}
		hosts = filteredHosts
	}

	// Filter by resource pool if specified
	if params.ResourcePool != nil && *params.ResourcePool != "" {
		filteredHosts := make([]VMHost, 0)
		for _, host := range hosts {
			if host.ResourcePool == *params.ResourcePool {
				filteredHosts = append(filteredHosts, host)
			}
		}
		hosts = filteredHosts
	}

	// Filter by type
	if params.Type != "" {
		filteredHosts := make([]VMHost, 0)
		for _, host := range hosts {
			if host.Type == params.Type {
				filteredHosts = append(filteredHosts, host)
			}
		}
		hosts = filteredHosts
	}

	return hosts
}

func (e *VMHostExtension) getMockComposedVM(params ComposeVMParams) *ComposedVM {
	return &ComposedVM{
		SystemID:         "vm-" + params.VMHostID + "-001",
		VMHostID:         params.VMHostID,
		AvailabilityZone: "zone-a", // Mock zone
		Status:           "Allocating",
	}
}

func (e *VMHostExtension) getMockVMHostResources(vmHostID string) *VMHostResources {
	return &VMHostResources{
		Total:     ResourceInfo{Cores: 16, Memory: 32768},
		Used:      ResourceInfo{Cores: 6, Memory: 12288},
		Available: ResourceInfo{Cores: 10, Memory: 20480},
	}
}

// buildNetworkConfigParams builds network configuration parameters for MAAS API
func (e *VMHostExtension) buildNetworkConfigParams(params maasclient.Params, netConfig *NetworkConfigParams) error {
	if netConfig == nil || len(netConfig.Interfaces) == 0 {
		return nil
	}

	// Process each interface configuration
	interfaceConfigs := make([]string, 0, len(netConfig.Interfaces))

	for _, iface := range netConfig.Interfaces {
		interfaceSpec := ""

		// Add interface name if specified
		if iface.Name != "" {
			interfaceSpec = fmt.Sprintf("interface=%s", iface.Name)
		}

		// Add bridge configuration
		if iface.Bridge != nil && *iface.Bridge != "" {
			if interfaceSpec != "" {
				interfaceSpec += ","
			}
			interfaceSpec += fmt.Sprintf("bridge=%s", *iface.Bridge)
		}

		// Add MAC address configuration
		if iface.MacAddress != nil && *iface.MacAddress != "" {
			if interfaceSpec != "" {
				interfaceSpec += ","
			}
			interfaceSpec += fmt.Sprintf("mac=%s", *iface.MacAddress)
		}

		// Add static IP configuration
		if iface.IPConfig != nil {
			staticConfig := iface.IPConfig

			if interfaceSpec != "" {
				interfaceSpec += ","
			}

			interfaceSpec += fmt.Sprintf("ip=%s", staticConfig.IPAddress)

			if staticConfig.Gateway != "" {
				interfaceSpec += fmt.Sprintf(",gateway=%s", staticConfig.Gateway)
			}

			if staticConfig.Subnet != "" {
				interfaceSpec += fmt.Sprintf(",subnet=%s", staticConfig.Subnet)
			}

			if len(staticConfig.DNSServers) > 0 {
				interfaceSpec += fmt.Sprintf(",dns=%s", strings.Join(staticConfig.DNSServers, ","))
			}
		}

		if interfaceSpec != "" {
			interfaceConfigs = append(interfaceConfigs, interfaceSpec)
		}
	}

	// Set the interfaces parameter if we have any configurations
	if len(interfaceConfigs) > 0 {
		params.Set("interfaces", strings.Join(interfaceConfigs, ";"))
	}

	return nil
}

// validateNetworkConfig validates network configuration parameters
func (e *VMHostExtension) validateNetworkConfig(netConfig *NetworkConfigParams) error {
	if netConfig == nil {
		return nil
	}

	if len(netConfig.Interfaces) == 0 {
		return fmt.Errorf("network configuration must specify at least one interface")
	}

	// Validate each interface configuration
	for i, iface := range netConfig.Interfaces {
		// Check for static IP configuration consistency
		if iface.IPConfig != nil {
			staticConfig := iface.IPConfig

			if staticConfig.IPAddress == "" {
				return fmt.Errorf("interface %d: IP address is required for static IP configuration", i)
			}

			if staticConfig.Gateway == "" {
				return fmt.Errorf("interface %d: gateway is required for static IP configuration", i)
			}

			if staticConfig.Subnet == "" {
				return fmt.Errorf("interface %d: subnet is required for static IP configuration", i)
			}
		}

		// Check MAC address format if specified
		if iface.MacAddress != nil && *iface.MacAddress != "" {
			// Basic MAC address validation
			macAddr := *iface.MacAddress
			if len(macAddr) != 17 || !isValidMACAddress(macAddr) {
				return fmt.Errorf("interface %d: invalid MAC address format: %s", i, macAddr)
			}
		}
	}

	return nil
}

// isValidMACAddress performs basic MAC address format validation
func isValidMACAddress(mac string) bool {
	// Simple validation for MAC address format (XX:XX:XX:XX:XX:XX)
	if len(mac) != 17 {
		return false
	}

	for i, c := range mac {
		if i%3 == 2 {
			// Every third character should be a colon
			if c != ':' {
				return false
			}
		} else {
			// Other characters should be hex digits
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}

	return true
}

// Conversion functions for network configuration types

// ConvertVMNetworkToClientParams converts service layer network config to client params
func ConvertVMNetworkToClientParams(serviceNetConfig *lxd.VMNetworkConfig) *NetworkConfigParams {
	if serviceNetConfig == nil {
		return nil
	}

	// Create default interface configuration
	interfaceConfig := InterfaceConfig{
		Name:       "eth0", // Default interface
		Bridge:     serviceNetConfig.Bridge,
		MacAddress: serviceNetConfig.MacAddress,
	}

	// Add static IP configuration if present
	if serviceNetConfig.StaticIPConfig != nil {
		interfaceConfig.IPConfig = ConvertServiceStaticIPToClient(serviceNetConfig.StaticIPConfig)

		// Use the interface name from static IP config if specified
		if serviceNetConfig.StaticIPConfig.Interface != "" {
			interfaceConfig.Name = serviceNetConfig.StaticIPConfig.Interface
		}
	}

	return &NetworkConfigParams{
		Interfaces: []InterfaceConfig{interfaceConfig},
	}
}

// ConvertServiceStaticIPToClient converts service layer static IP config to client params
func ConvertServiceStaticIPToClient(serviceStaticIP *lxd.StaticIPSpec) *StaticIPParams {
	if serviceStaticIP == nil {
		return nil
	}

	return &StaticIPParams{
		IPAddress:  serviceStaticIP.IPAddress,
		Gateway:    serviceStaticIP.Gateway,
		Subnet:     serviceStaticIP.Subnet,
		DNSServers: serviceStaticIP.DNSServers,
	}
}

// convertMaasVMHostsToLocal converts MAAS VMHost objects to local VMHost objects
func (e *VMHostExtension) convertMaasVMHostsToLocal(maasVMHosts []maasclient.VMHost) []VMHost {
	var result []VMHost
	for _, maasVMHost := range maasVMHosts {
		vmHost := VMHost{
			ID:               maasVMHost.SystemID(),
			Name:             maasVMHost.Name(),
			Type:             maasVMHost.Type(),
			AvailabilityZone: "",
			ResourcePool:     "",
			Tags:             []string{}, // TODO: Get tags from MAAS API when available
		}

		// Set zone and resource pool if available
		if zone := maasVMHost.Zone(); zone != nil {
			vmHost.AvailabilityZone = zone.Name()
		}
		if pool := maasVMHost.ResourcePool(); pool != nil {
			vmHost.ResourcePool = pool.Name()
		}

		// Set resource information
		vmHost.Resources = VMHostResources{
			Total: ResourceInfo{
				Cores:  maasVMHost.TotalCores(),
				Memory: maasVMHost.TotalMemory(),
			},
			Used: ResourceInfo{
				Cores:  maasVMHost.UsedCores(),
				Memory: maasVMHost.UsedMemory(),
			},
			Available: ResourceInfo{
				Cores:  maasVMHost.AvailableCores(),
				Memory: maasVMHost.AvailableMemory(),
			},
		}

		result = append(result, vmHost)
	}
	return result
}
