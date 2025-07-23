# MAAS VMHosts API Integration Guide

## Overview

This document outlines exactly where and how to integrate the MAAS VMHosts API when it becomes available in the upstream `maas-client-go` library.

## Integration Points

### 1. Client Library Extension (`pkg/maas/client/vmhost.go`)

**Current State**: Mock implementations with TODO comments
**Target State**: Full MAAS VMHosts API integration

#### A. Replace Mock Implementation in `QueryVMHosts`:

```go
// BEFORE (Current Mock):
func (e *VMHostExtension) QueryVMHosts(ctx context.Context, params VMHostQueryParams) ([]VMHost, error) {
    // TODO: Implement when MAAS client supports VMHosts API
    return e.getMockVMHosts(params), nil
}

// AFTER (With VMHosts API):
func (e *VMHostExtension) QueryVMHosts(ctx context.Context, params VMHostQueryParams) ([]VMHost, error) {
    queryParams := maasclient.ParamsBuilder()
    
    if params.Type != "" {
        queryParams.Set("type", params.Type)
    }
    if params.Zone != nil {
        queryParams.Set("zone", *params.Zone)
    }
    if params.ResourcePool != nil {
        queryParams.Set("pool", *params.ResourcePool)
    }
    if len(params.Tags) > 0 {
        queryParams.Set("tags", strings.Join(params.Tags, ","))
    }

    // Use the new VMHosts API
    vmHosts, err := e.client.VMHosts().Query(ctx, queryParams.Build())
    if err != nil {
        return nil, fmt.Errorf("failed to query VM hosts: %w", err)
    }

    return e.convertMaasVMHostsToLocal(vmHosts), nil
}
```

#### B. Replace Mock Implementation in `ComposeVM`:

```go
// BEFORE (Current Mock):
func (e *VMHostExtension) ComposeVM(ctx context.Context, params ComposeVMParams) (*ComposedVM, error) {
    // TODO: Implement when MAAS client supports VMHosts API
    return e.getMockComposedVM(params), nil
}

// AFTER (With VMHosts API):
func (e *VMHostExtension) ComposeVM(ctx context.Context, params ComposeVMParams) (*ComposedVM, error) {
    composeParams := maasclient.ParamsBuilder().
        Set("cores", strconv.Itoa(params.Cores)).
        Set("memory", strconv.Itoa(params.Memory))

    // Add existing disk, profile, project parameters
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

    // *** NEW: Add network configuration support ***
    if params.NetworkConfig != nil {
        if err := e.buildNetworkConfigParams(composeParams, params.NetworkConfig); err != nil {
            return nil, fmt.Errorf("failed to build network configuration: %w", err)
        }
    }

    // Use the new VMHosts API
    composedVM, err := e.client.VMHosts().VMHost(params.VMHostID).Composer().
        Compose(ctx, composeParams.Build())
    if err != nil {
        return nil, fmt.Errorf("failed to compose VM: %w", err)
    }

    return &ComposedVM{
        SystemID:         composedVM.SystemID(),
        VMHostID:         params.VMHostID,
        AvailabilityZone: composedVM.Zone().Name(),
        Status:           composedVM.Status(),
    }, nil
}
```

### 2. Service Layer Integration (`pkg/maas/lxd/service.go`)

#### A. Replace Mock VM Composition:

```go
// BEFORE (Current Mock):
func (s *Service) ComposeVM(ctx context.Context, vmSpec *VMSpec) (*LXDVMResult, error) {
    // ... host selection logic ...
    
    // TODO: Implement VM composition when MAAS client supports VMHosts API
    // For now, return a mock result for testing purposes
    mockSystemID := fmt.Sprintf("vm-%s-001", selectedHost.SystemID)
    composedVM := &mockComposedVM{systemID: mockSystemID}
    
    // ... rest of mock implementation ...
}

// AFTER (With VMHosts API):
func (s *Service) ComposeVM(ctx context.Context, vmSpec *VMSpec) (*LXDVMResult, error) {
    // ... existing host selection logic remains the same ...
    
    // Create VMHost extension for actual API calls
    vmHostExt := client.NewVMHostExtension(s.maasClient)
    
    // Convert VMSpec to client parameters
    clientParams := client.ComposeVMParams{
        VMHostID:      selectedHost.SystemID,
        Cores:         vmSpec.Cores,
        Memory:        vmSpec.Memory,
        Disks:         s.convertDisksToClientFormat(vmSpec.Disks),
        Profile:       vmSpec.Profile,
        Project:       vmSpec.Project,
        UserData:      vmSpec.UserData,
        Tags:          vmSpec.Tags,
        NetworkConfig: client.ConvertVMNetworkToClientParams(vmSpec.NetworkConfig),
    }
    
    // Compose VM using actual MAAS API
    composedVM, err := vmHostExt.ComposeVM(ctx, clientParams)
    if err != nil {
        return nil, WrapLXDError(err, LXDErrorVMCreationFailed, "failed to compose VM")
    }
    
    // Build result with actual data
    result := &LXDVMResult{
        SystemID:      composedVM.SystemID,
        HostID:        selectedHost.SystemID,
        FailureDomain: composedVM.AvailabilityZone,
        Project:       vmSpec.Project,
    }
    
    // Extract actual network status
    if err := s.updateVMResultWithNetworkStatus(ctx, result, vmSpec.NetworkConfig); err != nil {
        log.Error(err, "Failed to extract network status", "systemID", result.SystemID)
        result.NetworkStatus = s.buildNetworkStatus(vmSpec.NetworkConfig)
    }
    
    return result, nil
}
```

#### B. Replace Mock Host Discovery:

```go
// BEFORE (Current Mock):
func (s *Service) GetAvailableLXDHosts(ctx context.Context) ([]LXDHost, error) {
    // TODO: Query MAAS for LXD VM hosts when VMHosts API is available
    return e.getMockVMHosts(params), nil
}

// AFTER (With VMHosts API):
func (s *Service) GetAvailableLXDHosts(ctx context.Context) ([]LXDHost, error) {
    mm := s.scope.MaasMachine
    
    // Build query parameters
    queryParams := client.VMHostQueryParams{
        Type: "lxd",
    }
    
    if mm.Spec.ResourcePool != nil {
        queryParams.ResourcePool = mm.Spec.ResourcePool
    }
    
    failureDomain := mm.Spec.FailureDomain
    if failureDomain == nil {
        failureDomain = s.scope.Machine.Spec.FailureDomain
    }
    if failureDomain != nil {
        queryParams.Zone = failureDomain
    }
    
    queryParams.Tags = mm.Spec.Tags
    
    // Use VMHost extension to query actual hosts
    vmHostExt := client.NewVMHostExtension(s.maasClient)
    vmHosts, err := vmHostExt.QueryVMHosts(ctx, queryParams)
    if err != nil {
        return nil, WrapLXDError(err, LXDErrorHostUnavailable, "failed to query LXD hosts")
    }
    
    // Convert to LXDHost format
    hosts := make([]LXDHost, 0, len(vmHosts))
    for _, vmHost := range vmHosts {
        hosts = append(hosts, s.convertClientVMHostToLXDHost(vmHost))
    }
    
    return hosts, nil
}
```

### 3. Network Status Extraction Enhancement

#### A. Real Network Status Extraction:

```go
// AFTER (With VMHosts API):
func (s *Service) extractNetworkStatusFromVM(ctx context.Context, systemID string, requestedConfig *VMNetworkConfig) (*VMNetworkStatus, error) {
    // Get VM details with network interfaces
    vm, err := s.maasClient.Machines().Machine(systemID).Get(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get VM for network status: %w", err)
    }
    
    // Extract actual network interfaces (when API supports it)
    interfaces, err := vm.Interfaces().List(ctx)  // Future API method
    if err != nil {
        return nil, fmt.Errorf("failed to get VM interfaces: %w", err)
    }
    
    // Find the primary interface with assigned IP
    for _, iface := range interfaces {
        if iface.Type() == "ethernet" && len(iface.IPAddresses()) > 0 {
            status := &VMNetworkStatus{
                AssignedIP:   &iface.IPAddresses()[0].IPAddress,
                Interface:    &iface.Name(),
                MacAddress:   &iface.MACAddress(),
                ConfigMethod: s.detectConfigMethod(iface),
            }
            
            // Extract gateway and DNS if available
            if gw := iface.Gateway(); gw != nil {
                gwStr := gw.String()
                status.Gateway = &gwStr
            }
            
            return status, nil
        }
    }
    
    return &VMNetworkStatus{ConfigMethod: "dhcp"}, nil
}
```

## Required Changes to maas-client-go

The upstream `maas-client-go` library needs to add VMHosts API support:

### Expected API Structure:

```go
// In maas-client-go library
type ClientSetInterface interface {
    // Existing methods...
    Machines() MachinesInterface
    
    // NEW: VMHosts API
    VMHosts() VMHostsInterface  // <-- This needs to be added
}

type VMHostsInterface interface {
    Query(ctx context.Context, params Params) ([]VMHost, error)
    VMHost(systemID string) VMHostInterface
}

type VMHostInterface interface {
    Get(ctx context.Context) (VMHost, error)
    Composer() VMComposerInterface
    Machines() VMHostMachinesInterface
}

type VMComposerInterface interface {
    Compose(ctx context.Context, params Params) (Machine, error)
}

type VMHost interface {
    SystemID() string
    Name() string
    Type() string  // "lxd", "virsh", etc.
    Zone() Zone
    ResourcePool() ResourcePool
    // Resource information methods
    TotalCores() int
    TotalMemory() int
    UsedCores() int
    UsedMemory() int
    AvailableCores() int
    AvailableMemory() int
}
```

## Migration Steps

### Phase 1: Preparation (Before VMHosts API)
- ✅ Current state - Mock implementations ready
- ✅ All integration points identified with TODO comments
- ✅ Network configuration structures ready

### Phase 2: API Integration (When VMHosts API available)
1. **Update maas-client-go dependency** to version with VMHosts support
2. **Replace mock implementations** in `pkg/maas/client/vmhost.go`
3. **Update service layer** in `pkg/maas/lxd/service.go` 
4. **Add conversion functions** between client and service types
5. **Update tests** to use real API calls

### Phase 3: Network Enhancement (Advanced Integration)
1. **Add interface inspection** when MAAS supports VM interface details
2. **Enhance IP extraction** from actual VM network interfaces
3. **Add network configuration validation** against MAAS constraints

## Testing Strategy

### Current Testing (Mock-based):
```bash
go test ./pkg/maas/client/...
go test ./pkg/maas/lxd/...
```

### Future Testing (API-based):
```bash
# Integration tests against real MAAS
MAAS_ENDPOINT=http://maas.example.com/MAAS \
MAAS_APIKEY=your-key \
go test -tags integration ./pkg/maas/client/...
```

## File Checklist for Integration

When VMHosts API becomes available, update these files in order:

- [ ] `pkg/maas/client/vmhost.go` - Replace all TODO sections
- [ ] `pkg/maas/lxd/service.go` - Replace mock VM composition and host discovery  
- [ ] Add new conversion functions between client and service types
- [ ] Update tests to use real API endpoints
- [ ] Update documentation and examples

The current implementation is fully prepared for seamless integration once the upstream API becomes available.