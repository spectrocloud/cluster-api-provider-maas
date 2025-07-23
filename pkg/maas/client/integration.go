package client

import (
	"context"

	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// ClientExtensions provides extended functionality for MAAS client operations
// This acts as a bridge between the generic MAAS client and provider-specific operations
type ClientExtensions struct {
	client    maasclient.ClientSetInterface
	vmHostExt *VMHostExtension
}

// NewClientExtensions creates a new client extensions instance
func NewClientExtensions(client maasclient.ClientSetInterface) *ClientExtensions {
	return &ClientExtensions{
		client:    client,
		vmHostExt: NewVMHostExtension(client),
	}
}

// VMHostExtensionInterface defines the interface for VM host operations
// This allows for easier testing and future implementation flexibility
type VMHostExtensionInterface interface {
	QueryVMHosts(ctx context.Context, params VMHostQueryParams) ([]VMHost, error)
	ComposeVM(ctx context.Context, params ComposeVMParams) (*ComposedVM, error)
	DecomposeVM(ctx context.Context, vmHostID, systemID string) error
	GetVMHostResources(ctx context.Context, vmHostID string) (*VMHostResources, error)
}

// VMHosts returns the VM host extension
func (c *ClientExtensions) VMHosts() VMHostExtensionInterface {
	return c.vmHostExt
}

// LXDHostAdapter adapts VMHost to lxd.LXDHost for integration
type LXDHostAdapter struct {
	vmHost VMHost
}

// NewLXDHostAdapter creates an adapter from VMHost to LXDHost
func NewLXDHostAdapter(vmHost VMHost) *LXDHostAdapter {
	return &LXDHostAdapter{vmHost: vmHost}
}

// ToLXDHost converts VMHost to lxd.LXDHost
func (a *LXDHostAdapter) ToLXDHost() lxd.LXDHost {
	return lxd.LXDHost{
		SystemID:         a.vmHost.ID,
		Hostname:         a.vmHost.Name,
		AvailabilityZone: a.vmHost.AvailabilityZone,
		ResourcePool:     a.vmHost.ResourcePool,
		Available: lxd.ResourceInfo{
			Cores:  a.vmHost.Resources.Available.Cores,
			Memory: a.vmHost.Resources.Available.Memory,
		},
		Used: lxd.ResourceInfo{
			Cores:  a.vmHost.Resources.Used.Cores,
			Memory: a.vmHost.Resources.Used.Memory,
		},
		LXDCapabilities: lxd.LXDCapabilities{
			VMSupport:      true,
			Projects:       []string{"default", "maas"},
			Profiles:       []string{"default", "maas"},
			StoragePools:   []string{"default", "ssd-pool", "hdd-pool"},
			NetworkBridges: []string{"br0", "lxdbr0"},
		},
	}
}

// VMHostsToLXDHosts converts a slice of VMHost to a slice of lxd.LXDHost
func VMHostsToLXDHosts(vmHosts []VMHost) []lxd.LXDHost {
	lxdHosts := make([]lxd.LXDHost, 0, len(vmHosts))
	for _, vmHost := range vmHosts {
		adapter := NewLXDHostAdapter(vmHost)
		lxdHosts = append(lxdHosts, adapter.ToLXDHost())
	}
	return lxdHosts
}

// LXDVMSpecAdapter adapts lxd.VMSpec to ComposeVMParams
type LXDVMSpecAdapter struct {
	vmSpec *lxd.VMSpec
	hostID string
}

// NewLXDVMSpecAdapter creates an adapter from lxd.VMSpec to ComposeVMParams
func NewLXDVMSpecAdapter(vmSpec *lxd.VMSpec, hostID string) *LXDVMSpecAdapter {
	return &LXDVMSpecAdapter{vmSpec: vmSpec, hostID: hostID}
}

// ToComposeVMParams converts lxd.VMSpec to ComposeVMParams
func (a *LXDVMSpecAdapter) ToComposeVMParams() ComposeVMParams {
	params := ComposeVMParams{
		VMHostID: a.hostID,
		Cores:    a.vmSpec.Cores,
		Memory:   a.vmSpec.Memory,
		UserData: a.vmSpec.UserData,
		Tags:     a.vmSpec.Tags,
		Profile:  a.vmSpec.Profile,
		Project:  a.vmSpec.Project,
	}

	// Convert disks
	if len(a.vmSpec.Disks) > 0 {
		params.Disks = make([]DiskSpec, 0, len(a.vmSpec.Disks))
		for _, disk := range a.vmSpec.Disks {
			params.Disks = append(params.Disks, DiskSpec{
				Size: disk.Size,
				Pool: disk.Pool,
			})
		}
	}

	return params
}

// ComposedVMAdapter adapts ComposedVM to lxd.LXDVMResult
type ComposedVMAdapter struct {
	composedVM *ComposedVM
}

// NewComposedVMAdapter creates an adapter from ComposedVM to lxd.LXDVMResult
func NewComposedVMAdapter(composedVM *ComposedVM) *ComposedVMAdapter {
	return &ComposedVMAdapter{composedVM: composedVM}
}

// ToLXDVMResult converts ComposedVM to lxd.LXDVMResult
func (a *ComposedVMAdapter) ToLXDVMResult(project string) *lxd.LXDVMResult {
	result := &lxd.LXDVMResult{
		SystemID:      a.composedVM.SystemID,
		HostID:        a.composedVM.VMHostID,
		FailureDomain: a.composedVM.AvailabilityZone,
		Project:       project,
	}

	// Generate provider ID if we have availability zone
	if result.FailureDomain != "" {
		result.ProviderID = "maas://" + result.FailureDomain + "/" + result.SystemID
	} else {
		result.ProviderID = "maas:///" + result.SystemID
	}

	return result
}

// ClientExtensionFactory provides a factory for creating client extensions
// This allows for easier testing and dependency injection
type ClientExtensionFactory interface {
	CreateExtensions(client maasclient.ClientSetInterface) *ClientExtensions
}

// DefaultClientExtensionFactory is the default factory implementation
type DefaultClientExtensionFactory struct{}

// CreateExtensions creates new client extensions using the default implementation
func (f *DefaultClientExtensionFactory) CreateExtensions(client maasclient.ClientSetInterface) *ClientExtensions {
	return NewClientExtensions(client)
}

// GetDefaultClientExtensionFactory returns the default factory instance
func GetDefaultClientExtensionFactory() ClientExtensionFactory {
	return &DefaultClientExtensionFactory{}
}
