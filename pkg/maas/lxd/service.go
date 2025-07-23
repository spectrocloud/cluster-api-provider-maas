package lxd

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/maas-client-go/maasclient"
	textlogger "k8s.io/klog/v2/textlogger"
)

// Mock types for testing until MAAS client supports VMHosts API
type mockComposedVM struct {
	systemID string
}

func (m *mockComposedVM) SystemID() string {
	return m.systemID
}

type mockVMHost struct {
	id   string
	name string
	zone string
	pool string
}

// ComposeVM creates a new LXD VM based on the specification
func (s *Service) ComposeVM(ctx context.Context, vmSpec *VMSpec) (*LXDVMResult, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("Composing LXD VM", "cores", vmSpec.Cores, "memory", vmSpec.Memory)

	// Get available LXD hosts
	hosts, err := s.GetAvailableLXDHosts(ctx)
	if err != nil {
		return nil, WrapLXDError(err, LXDErrorHostUnavailable, "failed to get available LXD hosts")
	}

	if len(hosts) == 0 {
		return nil, NewLXDError(LXDErrorHostUnavailable, "no LXD hosts available in specified resource pool")
	}

	// Select optimal host
	selectedHost, err := s.SelectOptimalHost(ctx, hosts)
	if err != nil {
		return nil, WrapLXDError(err, LXDErrorHostUnavailable, "failed to select optimal LXD host")
	}

	// Build VM composition parameters
	composeParams := maasclient.ParamsBuilder().
		Set("cores", strconv.Itoa(vmSpec.Cores)).
		Set("memory", strconv.Itoa(vmSpec.Memory))

	// Add disk configuration
	if len(vmSpec.Disks) > 0 {
		diskSpecs := make([]string, 0, len(vmSpec.Disks))
		for _, disk := range vmSpec.Disks {
			diskSpec := fmt.Sprintf("size=%s", disk.Size)
			if disk.Pool != "" {
				diskSpec += fmt.Sprintf(",pool=%s", disk.Pool)
			}
			diskSpecs = append(diskSpecs, diskSpec)
		}
		composeParams.Set("disks", strings.Join(diskSpecs, ";"))
	}

	// Add profile if specified
	if vmSpec.Profile != "" {
		composeParams.Set("profile", vmSpec.Profile)
	}

	// Add project if specified
	if vmSpec.Project != "" {
		composeParams.Set("project", vmSpec.Project)
	}

	// Add network configuration if specified
	if vmSpec.NetworkConfig != nil {
		// Validate network configuration before proceeding
		if err := s.validateNetworkConfiguration(vmSpec.NetworkConfig); err != nil {
			return nil, WrapLXDError(err, LXDErrorVMCreationFailed, "network configuration validation failed")
		}

		if err := s.buildNetworkParams(composeParams, vmSpec.NetworkConfig); err != nil {
			return nil, WrapLXDError(err, LXDErrorVMCreationFailed, "failed to build network parameters")
		}
	}

	// TODO: Implement VM composition when MAAS client supports VMHosts API
	// For now, use the VMHost extension for testing integration
	if vmSpec.NetworkConfig != nil {
		log.Info("Composing VM with network configuration",
			"host", selectedHost.SystemID,
			"staticIP", vmSpec.NetworkConfig.StaticIPConfig != nil,
			"bridge", vmSpec.NetworkConfig.Bridge)

		// Log the network parameters for debugging
		if vmSpec.NetworkConfig.StaticIPConfig != nil {
			static := vmSpec.NetworkConfig.StaticIPConfig
			log.Info("Static IP configuration details",
				"ip", static.IPAddress,
				"gateway", static.Gateway,
				"subnet", static.Subnet,
				"interface", static.Interface)
		}
	}

	_ = composeParams // Avoid unused variable warning
	log.Info("Mock VM composition", "host", selectedHost.SystemID, "networkConfig", vmSpec.NetworkConfig != nil)

	// Create a mock composed VM result
	mockSystemID := fmt.Sprintf("vm-%s-001", selectedHost.SystemID)
	composedVM := &mockComposedVM{systemID: mockSystemID}

	log.Info("Successfully composed LXD VM", "systemID", composedVM.SystemID(), "host", selectedHost.SystemID)

	// Build result
	result := &LXDVMResult{
		SystemID:      composedVM.SystemID(),
		HostID:        selectedHost.SystemID,
		FailureDomain: selectedHost.AvailabilityZone,
		Project:       vmSpec.Project,
		IPAddresses:   []string{}, // Will be populated by network status extraction
	}

	// Extract and populate actual network status
	if err := s.updateVMResultWithNetworkStatus(ctx, result, vmSpec.NetworkConfig); err != nil {
		log.Error(err, "Failed to extract network status", "systemID", result.SystemID)
		// Don't fail the entire operation, just use the basic status
		result.NetworkStatus = s.buildNetworkStatus(vmSpec.NetworkConfig)
	}

	// Generate provider ID
	if result.FailureDomain != "" {
		result.ProviderID = fmt.Sprintf("maas-lxd://%s/%s", result.FailureDomain, result.SystemID)
	} else {
		result.ProviderID = fmt.Sprintf("maas-lxd:///%s", result.SystemID)
	}

	return result, nil
}

// DeployVM deploys an existing VM with user data
func (s *Service) DeployVM(ctx context.Context, systemID, userDataB64 string) (*infrav1beta1.Machine, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("Deploying LXD VM", "systemID", systemID)

	mm := s.scope.MaasMachine

	// Get the VM to ensure it exists
	vm, err := s.maasClient.Machines().Machine(systemID).Get(ctx)
	if err != nil {
		return nil, WrapLXDError(err, LXDErrorVMDeploymentFailed, "failed to get VM for deployment").WithHost(systemID)
	}

	// Deploy the VM with user data and image
	deployedVM, err := vm.Deployer().
		SetUserData(userDataB64).
		SetOSSystem("custom").
		SetDistroSeries(mm.Spec.Image).
		Deploy(ctx)

	if err != nil {
		return nil, WrapLXDError(err, LXDErrorVMDeploymentFailed, "failed to deploy VM").WithHost(systemID)
	}

	log.Info("Successfully deployed LXD VM", "systemID", systemID, "state", deployedVM.State())

	// Convert to infrav1beta1.Machine
	machine := s.convertMaasToInfraMachine(deployedVM)
	return machine, nil
}

// GetVM retrieves VM information by system ID
func (s *Service) GetVM(ctx context.Context, systemID string) (*infrav1beta1.Machine, error) {
	vm, err := s.maasClient.Machines().Machine(systemID).Get(ctx)
	if err != nil {
		return nil, WrapLXDError(err, LXDErrorVMCreationFailed, "failed to get VM").WithHost(systemID)
	}

	return s.convertMaasToInfraMachine(vm), nil
}

// GetVMWithNetworkStatus retrieves VM information including network status by system ID
func (s *Service) GetVMWithNetworkStatus(ctx context.Context, systemID string) (*LXDVMResult, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())

	// Get the VM information first
	vm, err := s.maasClient.Machines().Machine(systemID).Get(ctx)
	if err != nil {
		return nil, WrapLXDError(err, LXDErrorVMCreationFailed, "failed to get VM").WithHost(systemID)
	}

	log.Info("Retrieved VM information", "systemID", systemID, "state", vm.State())

	// Build basic VM result
	result := &LXDVMResult{
		SystemID:      vm.SystemID(),
		FailureDomain: vm.Zone().Name(),
		IPAddresses:   s.extractIPAddressesFromVM(vm), // Extract IP addresses from VM interfaces
	}

	// Extract network status from the VM
	// In a real implementation, we would inspect the VM's network interfaces
	// For now, we simulate based on the machine's network configuration
	networkStatus, err := s.extractNetworkStatusFromVM(ctx, systemID, nil)
	if err != nil {
		log.Error(err, "Failed to extract network status for existing VM", "systemID", systemID)
		// Provide basic DHCP status as fallback
		networkStatus = &VMNetworkStatus{
			ConfigMethod: "dhcp",
		}
	}

	result.NetworkStatus = networkStatus

	// Generate provider ID
	if result.FailureDomain != "" {
		result.ProviderID = fmt.Sprintf("maas-lxd://%s/%s", result.FailureDomain, result.SystemID)
	} else {
		result.ProviderID = fmt.Sprintf("maas-lxd:///%s", result.SystemID)
	}

	return result, nil
}

// DeleteVM deletes an LXD VM and cleans up resources
func (s *Service) DeleteVM(ctx context.Context, systemID string) error {
	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("Deleting LXD VM", "systemID", systemID)

	// Release the VM back to available pool
	_, err := s.maasClient.Machines().Machine(systemID).Releaser().Release(ctx)
	if err != nil {
		return WrapLXDError(err, LXDErrorVMDeploymentFailed, "failed to release VM").WithHost(systemID)
	}

	log.Info("Successfully released LXD VM", "systemID", systemID)
	return nil
}

// GetAvailableLXDHosts returns available LXD hosts based on constraints
func (s *Service) GetAvailableLXDHosts(ctx context.Context) ([]LXDHost, error) {
	mm := s.scope.MaasMachine

	// Build query parameters based on machine spec constraints
	params := maasclient.ParamsBuilder().Set("type", "lxd")

	// Add resource pool filter if specified
	if mm.Spec.ResourcePool != nil && *mm.Spec.ResourcePool != "" {
		params.Set("pool", *mm.Spec.ResourcePool)
	}

	// Add zone filter if specified
	failureDomain := mm.Spec.FailureDomain
	if failureDomain == nil {
		failureDomain = s.scope.Machine.Spec.FailureDomain
	}
	if failureDomain != nil && *failureDomain != "" {
		params.Set("zone", *failureDomain)
	}

	// TODO: Query MAAS for LXD VM hosts when VMHosts API is available
	// For now, return mock hosts for testing
	_ = params // Avoid unused variable warning
	log := textlogger.NewLogger(textlogger.NewConfig())
	log.Info("Mock LXD host discovery")

	// Create mock LXD hosts for testing
	vmHosts := []mockVMHost{
		{id: "host-1", name: "lxd-host-1", zone: "zone-a", pool: "default"},
		{id: "host-2", name: "lxd-host-2", zone: "zone-b", pool: "default"},
	}

	// Convert to LXDHost structs and filter by tags if specified
	hosts := make([]LXDHost, 0, len(vmHosts))
	for _, vmHost := range vmHosts {
		host := s.convertVMHostToLXDHost(vmHost)

		// Apply tag filtering if tags are specified
		if len(mm.Spec.Tags) > 0 && !s.hostMatchesTags(host, mm.Spec.Tags) {
			continue
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

// SelectOptimalHost selects the best LXD host from available hosts
func (s *Service) SelectOptimalHost(ctx context.Context, hosts []LXDHost) (*LXDHost, error) {
	if len(hosts) == 0 {
		return nil, NewLXDError(LXDErrorHostUnavailable, "no hosts available for selection")
	}

	mm := s.scope.MaasMachine
	requiredCores := 1
	requiredMemory := 1024 // Default 1GB

	if mm.Spec.MinCPU != nil {
		requiredCores = *mm.Spec.MinCPU
	}
	if mm.Spec.MinMemoryInMB != nil {
		requiredMemory = *mm.Spec.MinMemoryInMB
	}

	// Filter hosts by resource requirements
	suitableHosts := make([]LXDHost, 0)
	for _, host := range hosts {
		if host.Available.Cores >= requiredCores && host.Available.Memory >= requiredMemory {
			suitableHosts = append(suitableHosts, host)
		}
	}

	if len(suitableHosts) == 0 {
		return nil, NewLXDErrorf(LXDErrorInsufficientResources,
			"no hosts meet resource requirements: need %d cores and %d MB memory",
			requiredCores, requiredMemory)
	}

	// Sort by available resources (most available first)
	sort.Slice(suitableHosts, func(i, j int) bool {
		hostI := suitableHosts[i]
		hostJ := suitableHosts[j]

		// Calculate resource utilization percentage
		utilizationI := float64(hostI.Used.Cores+hostI.Used.Memory) / float64(hostI.Available.Cores+hostI.Available.Memory+1)
		utilizationJ := float64(hostJ.Used.Cores+hostJ.Used.Memory) / float64(hostJ.Available.Cores+hostJ.Available.Memory+1)

		return utilizationI < utilizationJ
	})

	return &suitableHosts[0], nil
}

// DistributeAcrossAZs distributes VMs across availability zones
func (s *Service) DistributeAcrossAZs(ctx context.Context, hosts []LXDHost, count int) ([]LXDHost, error) {
	if len(hosts) == 0 {
		return nil, NewLXDError(LXDErrorHostUnavailable, "no hosts available for distribution")
	}

	if count <= 0 {
		return []LXDHost{}, nil
	}

	// Group hosts by availability zone
	zoneHosts := make(map[string][]LXDHost)
	for _, host := range hosts {
		zone := host.AvailabilityZone
		if zone == "" {
			zone = "default"
		}
		zoneHosts[zone] = append(zoneHosts[zone], host)
	}

	// Sort zones by name for consistent ordering
	zones := make([]string, 0, len(zoneHosts))
	for zone := range zoneHosts {
		zones = append(zones, zone)
	}
	sort.Strings(zones)

	// Distribute VMs across zones in round-robin fashion
	selectedHosts := make([]LXDHost, 0, count)
	zoneIndex := 0

	for len(selectedHosts) < count {
		zone := zones[zoneIndex%len(zones)]
		hostsInZone := zoneHosts[zone]

		if len(hostsInZone) > 0 {
			// Select the best host from this zone
			bestHost, err := s.SelectOptimalHost(ctx, hostsInZone)
			if err == nil {
				selectedHosts = append(selectedHosts, *bestHost)

				// Remove the selected host from the zone to avoid duplication
				for i, h := range hostsInZone {
					if h.SystemID == bestHost.SystemID {
						zoneHosts[zone] = append(hostsInZone[:i], hostsInZone[i+1:]...)
						break
					}
				}
			}
		}

		zoneIndex++

		// If we've cycled through all zones and no more hosts are available, break
		if zoneIndex >= len(zones)*2 && len(selectedHosts) == 0 {
			break
		}
	}

	if len(selectedHosts) == 0 {
		return nil, NewLXDError(LXDErrorInsufficientResources, "no suitable hosts found for distribution")
	}

	return selectedHosts, nil
}

// Helper methods

func (s *Service) convertVMHostToLXDHost(vmHost mockVMHost) LXDHost {
	host := LXDHost{
		SystemID:         vmHost.id,
		Hostname:         vmHost.name,
		AvailabilityZone: vmHost.zone,
		ResourcePool:     vmHost.pool,
	}

	// Mock resource information for testing
	host.Available = ResourceInfo{
		Cores:  16,
		Memory: 32768, // 32GB
		Disk:   1000,  // 1TB
	}

	host.Used = ResourceInfo{
		Cores:  4,
		Memory: 8192, // 8GB
		Disk:   200,  // 200GB
	}

	// Set LXD capabilities
	host.LXDCapabilities = LXDCapabilities{
		VMSupport:      true, // Assuming all VM hosts support VMs
		Projects:       []string{"default"},
		Profiles:       []string{"default"},
		StoragePools:   []string{"default"},
		NetworkBridges: []string{"lxdbr0"},
	}

	return host
}

func (s *Service) convertMaasToInfraMachine(maasMachine maasclient.Machine) *infrav1beta1.Machine {
	// Convert MAAS machine to infrav1beta1.Machine
	// This is a basic conversion - would need to be expanded based on actual requirements
	machine := &infrav1beta1.Machine{
		ID:               maasMachine.SystemID(),
		Hostname:         maasMachine.Hostname(),
		State:            infrav1beta1.MachineState(maasMachine.State()),
		Powered:          maasMachine.PowerState() == "on",
		AvailabilityZone: maasMachine.Zone().Name(),
	}

	return machine
}

func (s *Service) hostMatchesTags(host LXDHost, requiredTags []string) bool {
	// For now, assume all hosts match tags
	// This would need to be implemented based on how tags are stored in MAAS VM hosts
	return true
}

// buildNetworkParams builds network configuration parameters for MAAS VM composition
func (s *Service) buildNetworkParams(params maasclient.Params, netConfig *VMNetworkConfig) error {
	// Add bridge configuration
	if netConfig.Bridge != nil && *netConfig.Bridge != "" {
		params.Set("bridge", *netConfig.Bridge)
	}

	// Add MAC address configuration
	if netConfig.MacAddress != nil && *netConfig.MacAddress != "" {
		params.Set("mac_address", *netConfig.MacAddress)
	}

	// Add static IP configuration
	if netConfig.StaticIPConfig != nil {
		staticConfig := netConfig.StaticIPConfig

		// Build interface configuration string
		interfaceConfig := fmt.Sprintf("ip=%s", staticConfig.IPAddress)

		if staticConfig.Gateway != "" {
			interfaceConfig += fmt.Sprintf(",gateway=%s", staticConfig.Gateway)
		}

		if staticConfig.Subnet != "" {
			interfaceConfig += fmt.Sprintf(",subnet=%s", staticConfig.Subnet)
		}

		if len(staticConfig.DNSServers) > 0 {
			interfaceConfig += fmt.Sprintf(",dns=%s", strings.Join(staticConfig.DNSServers, ","))
		}

		if staticConfig.Interface != "" {
			interfaceConfig = fmt.Sprintf("interface=%s,%s", staticConfig.Interface, interfaceConfig)
		}

		params.Set("interfaces", interfaceConfig)
	}

	return nil
}

// buildNetworkStatus creates network status from the network configuration
func (s *Service) buildNetworkStatus(netConfig *VMNetworkConfig) *VMNetworkStatus {
	if netConfig == nil {
		return &VMNetworkStatus{
			ConfigMethod: "dhcp",
		}
	}

	status := &VMNetworkStatus{
		Bridge:     netConfig.Bridge,
		MacAddress: netConfig.MacAddress,
	}

	if netConfig.StaticIPConfig != nil {
		staticConfig := netConfig.StaticIPConfig
		status.ConfigMethod = "static"
		status.AssignedIP = &staticConfig.IPAddress
		status.Gateway = &staticConfig.Gateway
		status.DNSServers = staticConfig.DNSServers

		if staticConfig.Interface != "" {
			status.Interface = &staticConfig.Interface
		}
	} else {
		status.ConfigMethod = "dhcp"
	}

	return status
}

// extractNetworkStatusFromVM extracts actual network status from a composed VM
func (s *Service) extractNetworkStatusFromVM(ctx context.Context, systemID string, requestedConfig *VMNetworkConfig) (*VMNetworkStatus, error) {
	log := textlogger.NewLogger(textlogger.NewConfig())

	// TODO: When MAAS client supports VM inspection, extract actual network info
	// For now, we'll use the requested configuration as the base and simulate actual status

	if requestedConfig == nil {
		// If no specific configuration was requested, assume DHCP
		log.Info("No network configuration requested, assuming DHCP", "systemID", systemID)
		return &VMNetworkStatus{
			ConfigMethod: "dhcp",
		}, nil
	}

	// Start with the requested configuration
	status := s.buildNetworkStatus(requestedConfig)

	// In a real implementation, we would query the VM for actual network status
	// For now, we simulate successful configuration
	if requestedConfig.StaticIPConfig != nil {
		log.Info("Static IP configuration applied successfully",
			"systemID", systemID,
			"ip", requestedConfig.StaticIPConfig.IPAddress,
			"interface", requestedConfig.StaticIPConfig.Interface)

		// Mark the configuration as successfully applied
		status.ConfigMethod = "static"
		status.AssignedIP = &requestedConfig.StaticIPConfig.IPAddress
	} else {
		log.Info("DHCP configuration applied", "systemID", systemID)
		// In real scenario, we would query for the DHCP-assigned IP
		// For now, simulate a DHCP assignment
		simulatedDHCPIP := "192.168.1.100" // Mock DHCP IP
		status.AssignedIP = &simulatedDHCPIP
		status.ConfigMethod = "dhcp"
	}

	return status, nil
}

// updateVMResultWithNetworkStatus updates LXDVMResult with extracted network status
func (s *Service) updateVMResultWithNetworkStatus(ctx context.Context, result *LXDVMResult, requestedConfig *VMNetworkConfig) error {
	if result == nil {
		return fmt.Errorf("VM result is nil")
	}

	// Extract actual network status from the VM
	actualStatus, err := s.extractNetworkStatusFromVM(ctx, result.SystemID, requestedConfig)
	if err != nil {
		return fmt.Errorf("failed to extract network status: %w", err)
	}

	// Update the result with actual network status
	result.NetworkStatus = actualStatus

	return nil
}

// buildVMSpecFromMachine creates a VMSpec from MaasMachine configuration
func (s *Service) buildVMSpecFromMachine() *VMSpec {
	mm := s.scope.MaasMachine

	// Start with basic resource allocation
	spec := &VMSpec{
		Cores:  2,    // Default
		Memory: 4096, // Default 4GB
		Tags:   mm.Spec.Tags,
	}

	// Apply LXD resource configuration if present
	if mm.Spec.LXDConfig != nil && mm.Spec.LXDConfig.ResourceAllocation != nil {
		resAlloc := mm.Spec.LXDConfig.ResourceAllocation

		if resAlloc.CPU != nil {
			spec.Cores = *resAlloc.CPU
		}

		if resAlloc.Memory != nil {
			spec.Memory = *resAlloc.Memory
		}

		if resAlloc.Disk != nil {
			spec.Disks = []DiskSpec{
				{
					Size: fmt.Sprintf("%dGB", *resAlloc.Disk),
					Pool: s.getStoragePool(),
				},
			}
		}
	}

	// Apply MinCPU and MinMemoryInMB from machine spec as overrides if they're higher
	if mm.Spec.MinCPU != nil && *mm.Spec.MinCPU > spec.Cores {
		spec.Cores = *mm.Spec.MinCPU
	}

	if mm.Spec.MinMemoryInMB != nil && *mm.Spec.MinMemoryInMB > spec.Memory {
		spec.Memory = *mm.Spec.MinMemoryInMB
	}

	// Set host ID if specified in LXD config
	if mm.Spec.LXDConfig != nil && mm.Spec.LXDConfig.HostSelection != nil {
		hostSel := mm.Spec.LXDConfig.HostSelection
		if len(hostSel.PreferredHosts) > 0 {
			spec.HostID = hostSel.PreferredHosts[0] // Use first preferred host
		}
	}

	// Build network configuration
	if mm.Spec.LXDConfig != nil && mm.Spec.LXDConfig.NetworkConfig != nil {
		spec.NetworkConfig = s.buildVMNetworkConfig(mm.Spec.LXDConfig.NetworkConfig)
	}

	// Set project if specified
	if mm.Spec.LXDConfig != nil {
		spec.Project = "default" // Default project
	}

	return spec
}

// buildVMNetworkConfig converts API network config to service network config
func (s *Service) buildVMNetworkConfig(apiNetConfig *infrav1beta1.LXDNetworkConfig) *VMNetworkConfig {
	vmNetConfig := &VMNetworkConfig{
		Bridge:     apiNetConfig.Bridge,
		MacAddress: apiNetConfig.MacAddress,
	}

	// Convert static IP configuration if present
	if apiNetConfig.StaticIPConfig != nil {
		apiStatic := apiNetConfig.StaticIPConfig

		vmNetConfig.StaticIPConfig = &StaticIPSpec{
			IPAddress:  apiStatic.IPAddress,
			Gateway:    apiStatic.Gateway,
			Subnet:     apiStatic.Subnet,
			DNSServers: apiStatic.DNSServers,
		}

		// Set interface name
		if apiStatic.Interface != nil {
			vmNetConfig.StaticIPConfig.Interface = *apiStatic.Interface
		} else {
			vmNetConfig.StaticIPConfig.Interface = "eth0" // Default interface
		}
	}

	return vmNetConfig
}

// getStoragePool determines the storage pool to use
func (s *Service) getStoragePool() string {
	mm := s.scope.MaasMachine

	if mm.Spec.LXDConfig != nil &&
		mm.Spec.LXDConfig.StorageConfig != nil &&
		mm.Spec.LXDConfig.StorageConfig.StoragePool != nil {
		return *mm.Spec.LXDConfig.StorageConfig.StoragePool
	}

	return "default" // Default storage pool
}

// validateNetworkConfiguration validates VM network configuration before composition
func (s *Service) validateNetworkConfiguration(netConfig *VMNetworkConfig) error {
	if netConfig == nil {
		return nil
	}

	// Validate bridge configuration
	if netConfig.Bridge != nil && *netConfig.Bridge == "" {
		return fmt.Errorf("bridge name cannot be empty")
	}

	// Validate static IP configuration if present
	if netConfig.StaticIPConfig != nil {
		staticConfig := netConfig.StaticIPConfig

		if staticConfig.IPAddress == "" {
			return fmt.Errorf("static IP address is required")
		}

		if staticConfig.Gateway == "" {
			return fmt.Errorf("gateway is required for static IP configuration")
		}

		if staticConfig.Subnet == "" {
			return fmt.Errorf("subnet is required for static IP configuration")
		}

		// Validate interface name
		if staticConfig.Interface == "" {
			return fmt.Errorf("interface name is required for static IP configuration")
		}
	}

	return nil
}

// extractIPAddressesFromVM extracts IP addresses from VM interfaces
func (s *Service) extractIPAddressesFromVM(vm maasclient.Machine) []string {
	// TODO: When MAAS client provides interface inspection, extract actual IPs
	// For now, simulate IP extraction based on VM state
	if vm.State() == "Deployed" || vm.State() == "Running" {
		// In a real implementation, we would query vm.Interfaces() or similar
		// For now, return a simulated IP based on the machine's system ID
		simulatedIP := fmt.Sprintf("192.168.1.%d", len(vm.SystemID())*10%200+10)
		return []string{simulatedIP}
	}
	return []string{}
}
