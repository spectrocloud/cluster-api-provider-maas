package lxd

import (
	"context"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// Service provides LXD VM provisioning operations following existing service patterns
type Service struct {
	scope      *scope.MachineScope
	maasClient maasclient.ClientSetInterface
}

// NewService creates a new LXD service instance following existing service patterns
func NewService(machineScope *scope.MachineScope) *Service {
	return &Service{
		scope:      machineScope,
		maasClient: scope.NewMaasClient(machineScope.ClusterScope),
	}
}

// VMSpec defines the specification for creating an LXD VM
type VMSpec struct {
	// Cores is the number of CPU cores to allocate
	Cores int
	// Memory is the amount of memory in MB to allocate
	Memory int
	// Disks defines the disk configuration for the VM
	Disks []DiskSpec
	// Profile is the LXD profile to apply
	Profile string
	// Project is the LXD project where the VM will be created
	Project string
	// UserData is the cloud-init user data for the VM
	UserData string
	// HostID is the preferred LXD host system ID
	HostID string
	// Tags for resource constraints and placement
	Tags []string
	// NetworkConfig defines the network configuration for the VM
	NetworkConfig *VMNetworkConfig
}

// DiskSpec defines disk configuration for LXD VMs
type DiskSpec struct {
	// Size of the disk (e.g., "20GB")
	Size string
	// Pool is the storage pool to use
	Pool string
}

// VMNetworkConfig defines network configuration for LXD VMs
type VMNetworkConfig struct {
	// Bridge specifies which bridge to connect the VM to
	Bridge *string
	// MacAddress allows specifying a custom MAC address
	MacAddress *string
	// StaticIPConfig defines static IP configuration for the VM
	StaticIPConfig *StaticIPSpec
}

// StaticIPSpec defines static IP assignment configuration for LXD VMs
type StaticIPSpec struct {
	// IPAddress is the static IP address to assign to the VM
	IPAddress string
	// Gateway is the default gateway for the static IP configuration
	Gateway string
	// Subnet defines the subnet mask or CIDR notation for the static IP
	Subnet string
	// DNSServers is a list of DNS server IP addresses
	DNSServers []string
	// Interface specifies the network interface to configure with static IP
	Interface string
}

// LXDVMResult contains the result of VM composition and deployment
type LXDVMResult struct {
	// SystemID is the MAAS system ID of the created VM
	SystemID string
	// HostID is the system ID of the LXD host
	HostID string
	// ProviderID is the cluster-api provider ID
	ProviderID string
	// FailureDomain is the availability zone of the VM
	FailureDomain string
	// IPAddresses contains the assigned IP addresses
	IPAddresses []string
	// Project is the LXD project name
	Project string
	// NetworkStatus contains the actual network configuration of the VM
	NetworkStatus *VMNetworkStatus
}

// VMNetworkStatus represents the actual network configuration status of an LXD VM
type VMNetworkStatus struct {
	// AssignedIP is the actual IP address assigned to the VM
	AssignedIP *string
	// Interface is the network interface name that has the assigned IP
	Interface *string
	// ConfigMethod indicates how the IP was configured ("dhcp" or "static")
	ConfigMethod string
	// Gateway is the gateway IP address being used
	Gateway *string
	// DNSServers is the list of DNS servers configured for the VM
	DNSServers []string
	// Bridge is the network bridge the VM is connected to
	Bridge *string
	// MacAddress is the MAC address assigned to the VM's interface
	MacAddress *string
}

// LXDHost represents an LXD-capable host with resource information
type LXDHost struct {
	// SystemID is the MAAS system ID of the host
	SystemID string
	// Hostname is the host's hostname
	Hostname string
	// AvailabilityZone is the failure domain/zone of the host
	AvailabilityZone string
	// ResourcePool is the MAAS resource pool the host belongs to
	ResourcePool string
	// Available resources on the host
	Available ResourceInfo
	// Used resources on the host
	Used ResourceInfo
	// LXDCapabilities describes the LXD capabilities of the host
	LXDCapabilities LXDCapabilities
}

// ResourceInfo contains resource allocation information
type ResourceInfo struct {
	// Cores is the number of CPU cores
	Cores int
	// Memory is the amount of memory in MB
	Memory int
	// Disk is the amount of disk space in GB
	Disk int
}

// LXDCapabilities describes the capabilities of an LXD host
type LXDCapabilities struct {
	// VMSupport indicates if the host supports VMs
	VMSupport bool
	// Projects are the available LXD projects
	Projects []string
	// Profiles are the available LXD profiles
	Profiles []string
	// StoragePools are the available storage pools
	StoragePools []string
	// NetworkBridges are the available network bridges
	NetworkBridges []string
}

// ServiceInterface defines the interface for LXD operations
type ServiceInterface interface {
	// ComposeVM creates a new LXD VM based on the specification
	ComposeVM(ctx context.Context, vmSpec *VMSpec) (*LXDVMResult, error)

	// DeployVM deploys an existing VM with user data
	DeployVM(ctx context.Context, systemID, userDataB64 string) (*infrav1beta1.Machine, error)

	// GetVM retrieves VM information by system ID
	GetVM(ctx context.Context, systemID string) (*infrav1beta1.Machine, error)

	// DeleteVM deletes an LXD VM and cleans up resources
	DeleteVM(ctx context.Context, systemID string) error

	// GetAvailableLXDHosts returns available LXD hosts based on constraints
	GetAvailableLXDHosts(ctx context.Context) ([]LXDHost, error)

	// SelectOptimalHost selects the best LXD host from available hosts
	SelectOptimalHost(ctx context.Context, hosts []LXDHost) (*LXDHost, error)

	// DistributeAcrossAZs distributes VMs across availability zones
	DistributeAcrossAZs(ctx context.Context, hosts []LXDHost, count int) ([]LXDHost, error)
}
