# MAAS Client Extensions

This package provides extensions to the MAAS client library to support VM host operations, specifically for LXD VM provisioning. These extensions are designed to integrate with the MAAS VMHosts API once it becomes available in the MAAS client library.

## Overview

The cluster-api-provider-maas currently uses bare metal provisioning through the MAAS client. To support LXD VM provisioning for control plane nodes, we need additional API operations that work with VM hosts (LXD hosts) for VM composition and decomposition.

## Architecture

### Core Components

1. **VMHostExtension** (`vmhost.go`): Provides VM host query, composition, and resource management operations
2. **ClientExtensions** (`integration.go`): Integrates extensions with the main MAAS client and provides adapter layers
3. **Adapter Classes**: Convert between different data structures (VMHost ↔ LXDHost, VMSpec ↔ ComposeVMParams)

### Key Interfaces

```go
type VMHostExtensionInterface interface {
    QueryVMHosts(ctx context.Context, params VMHostQueryParams) ([]VMHost, error)
    ComposeVM(ctx context.Context, params ComposeVMParams) (*ComposedVM, error)
    DecomposeVM(ctx context.Context, vmHostID, systemID string) error
    GetVMHostResources(ctx context.Context, vmHostID string) (*VMHostResources, error)
}
```

## Current Implementation Status

### Mock Implementation
Currently, all extensions use mock implementations since the MAAS client library does not yet support VMHosts API. The mock implementations:

- Return realistic test data for VM hosts with different configurations
- Provide proper resource allocation and utilization information
- Support filtering by availability zone, resource pool, and type
- Generate appropriate system IDs and provider IDs for composed VMs

### Future Integration
When the MAAS client library adds VMHosts API support, the implementation will be updated to:

1. Replace mock calls with actual MAAS API calls
2. Use proper MAAS VMHost objects and operations
3. Integrate with real VM composition and decomposition workflows
4. Support all MAAS VMHost features including profiles, projects, and storage pools

## Usage

### Basic VM Host Query
```go
// Create client extensions
extensions := NewClientExtensions(maasClient)

// Query LXD hosts in a specific zone
params := VMHostQueryParams{
    Type: "lxd",
    Zone: pointer.String("zone-a"),
    ResourcePool: pointer.String("compute-pool"),
}

hosts, err := extensions.VMHosts().QueryVMHosts(ctx, params)
```

### VM Composition
```go
// Compose a new VM on a specific host
composeParams := ComposeVMParams{
    VMHostID: "lxd-host-1",
    Cores:    4,
    Memory:   8192,
    Profile:  "maas",
    Project:  "default",
    Disks: []DiskSpec{
        {Size: "20GB", Pool: "default"},
    },
}

composedVM, err := extensions.VMHosts().ComposeVM(ctx, composeParams)
```

### Integration with LXD Service

The LXD service (`pkg/maas/lxd`) uses these extensions through adapter classes:

```go
// Convert VMHosts to LXDHosts for service layer
lxdHosts := VMHostsToLXDHosts(vmHosts)

// Convert LXD VMSpec to compose parameters
adapter := NewLXDVMSpecAdapter(vmSpec, selectedHostID)
composeParams := adapter.ToComposeVMParams()

// Convert composed VM result back to LXD format
resultAdapter := NewComposedVMAdapter(composedVM)
lxdResult := resultAdapter.ToLXDVMResult(project)
```

## Testing

All extensions include comprehensive test suites:

- **Unit Tests**: Test individual methods and data transformations
- **Integration Tests**: Test adapter layers and data flow between components
- **Mock Data Validation**: Ensure mock implementations provide realistic scenarios

### Running Tests

```bash
# Run all client extension tests
go test ./pkg/maas/client/...

# Run with coverage
go test -cover ./pkg/maas/client/...

# Run specific test suites
go test ./pkg/maas/client/ -run TestVMHostExtension
go test ./pkg/maas/client/ -run TestClientExtensions
```

## Error Handling

The extensions integrate with the LXD error handling system:

- **Retryable Errors**: Host unavailable, insufficient resources, temporary failures
- **Non-retryable Errors**: Profile not found, project not found, configuration errors
- **Error Wrapping**: All errors are properly wrapped with context and error types

## Mock Data Structure

The mock implementation provides realistic data for testing:

### Mock VM Hosts
- **lxd-host-1**: 16 cores, 32GB RAM, zone-a, default pool
- **lxd-host-2**: 16 cores, 32GB RAM, zone-b, default pool  
- **lxd-host-3**: 32 cores, 64GB RAM, zone-c, compute-pool (high-memory tagged)

### Mock Resource Utilization
Each host has realistic resource utilization:
- Total resources (hardware capacity)
- Used resources (currently allocated)
- Available resources (total - used)

## Future Development

### Phase 1: MAAS Client Library Integration
When VMHosts API becomes available:
1. Replace mock implementations with real MAAS API calls
2. Add proper error handling for MAAS API errors
3. Implement VM composition with actual LXD integration
4. Support advanced features like storage pools and network configuration

### Phase 2: Enhanced Features
Additional features to implement:
1. VM migration between hosts
2. Advanced placement policies (anti-affinity, resource balancing)
3. Integration with MAAS VM host health monitoring
4. Support for different virtualization backends (KVM, LXD)

### Phase 3: Performance Optimization
Performance improvements:
1. Caching of VM host resource information
2. Batch operations for multiple VM composition
3. Optimized host selection algorithms
4. Integration with MAAS resource monitoring

## Dependencies

- **MAAS Client**: `github.com/spectrocloud/maas-client-go/maasclient`
- **LXD Package**: `./pkg/maas/lxd` (for data type integration)
- **Testing**: Standard Go testing package + `k8s.io/utils/pointer` for test utilities

## Configuration

The extensions respect the same configuration as the main MAAS client:
- MAAS endpoint URL
- Authentication credentials  
- Connection timeouts and retries
- Resource filtering and constraints

When VMHosts API becomes available, additional configuration options will include:
- VM host selection policies
- Resource allocation strategies
- Storage pool preferences
- Network configuration options