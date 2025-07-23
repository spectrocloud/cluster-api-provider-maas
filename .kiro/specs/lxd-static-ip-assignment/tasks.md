# Implementation Plan

- [ ] 1. Extend API types for static IP configuration
  - Add `LXDStaticIPConfig` and `LXDNetworkStatus` structs to `maasmachine_types.go`
  - Update `LXDNetworkConfig` to include `StaticIPConfig` field
  - Add `NetworkStatus` field to `LXDMachineStatus` struct
  - Add appropriate JSON tags and validation annotations
  - _Requirements: 1.1, 1.2, 6.2_

- [ ] 2. Implement webhook validation for static IP configuration
  - [ ] 2.1 Add IP address format validation function
    - Create `validateIPAddress()` function for IPv4/IPv6 format checking
    - Implement subnet validation for CIDR notation
    - Add gateway IP validation logic
    - Write unit tests for IP format validation functions
    - _Requirements: 1.1, 4.1_

  - [ ] 2.2 Implement IP conflict detection
    - Create `checkIPConflicts()` function to scan existing MaasMachine resources
    - Add validation logic in `validateLXDConfig()` method
    - Implement subnet range validation
    - Write unit tests for conflict detection logic
    - _Requirements: 1.4, 4.2_

  - [ ] 2.3 Add static IP configuration validation to webhook
    - Integrate static IP validation into existing `maasmachine_webhook.go`
    - Add DNS server validation logic
    - Implement interface name validation
    - Write comprehensive webhook validation tests
    - _Requirements: 2.1, 2.2, 2.3, 5.1_

- [ ] 3. Extend LXD service interfaces for network configuration
  - [ ] 3.1 Update VMSpec structure with network configuration
    - Add `NetworkConfig` field to `VMSpec` struct in `interfaces.go`
    - Create `VMNetworkConfig` and `StaticIPSpec` structs
    - Update `LXDVMResult` to include network status information
    - Write unit tests for new data structures
    - _Requirements: 6.1_

  - [ ] 3.2 Enhance buildVMSpec method for static IP support
    - Modify `buildVMSpec()` in LXD service to include network configuration
    - Add logic to extract static IP config from MaasMachine spec
    - Implement fallback to DHCP when static IP not configured
    - Write unit tests for VMSpec building with network config
    - _Requirements: 3.1, 3.2, 5.2_

- [ ] 4. Extend MAAS client for static IP parameters
  - [ ] 4.1 Create network configuration parameter structures
    - Add `NetworkConfigParams`, `InterfaceConfig`, and `StaticIPParams` structs to client layer
    - Update `ComposeVMParams` to include network configuration
    - Create conversion functions between service and client types
    - Write unit tests for parameter structure conversion
    - _Requirements: 6.1_

  - [ ] 4.2 Implement MAAS client network configuration integration
    - Extend `ComposeVM` method to accept network configuration parameters
    - Add logic to transform static IP config into MAAS API parameters
    - Implement error handling for network configuration failures
    - Write integration tests for MAAS client network operations
    - _Requirements: 1.3, 6.3_

- [ ] 5. Update LXD service for static IP deployment
  - [ ] 5.1 Enhance ComposeVM method for network configuration
    - Modify `ComposeVM()` in `service.go` to handle static IP configuration
    - Add network validation before VM composition
    - Implement error handling for network configuration failures
    - Write unit tests for VM composition with static networking
    - _Requirements: 1.3, 4.3_

  - [ ] 5.2 Add network status extraction and reporting
    - Create function to extract network status from composed VM
    - Update VM result structures to include assigned network configuration
    - Implement logic to populate `LXDNetworkStatus` in machine status
    - Write unit tests for network status extraction
    - _Requirements: 6.3_

- [ ] 6. Update machine scope for network status management
  - [ ] 6.1 Add network status methods to machine scope
    - Create `SetLXDNetworkStatus()` method in `machine.go`
    - Add `GetLXDNetworkStatus()` method for status retrieval
    - Implement network configuration helper methods
    - Write unit tests for network status management methods
    - _Requirements: 6.2, 6.3_

  - [ ] 6.2 Integrate network status updates in controller
    - Modify machine controller to update network status after VM deployment
    - Add network-specific condition handling for deployment failures
    - Implement status updates for static IP assignment results
    - Write unit tests for controller network status integration
    - _Requirements: 6.3, 6.4_

- [ ] 7. Add condition constants for network operations
  - Extend `condition_consts.go` with network-specific condition constants
  - Add reasons for network configuration success and failure states
  - Include network validation failure reasons
  - Write unit tests to verify condition constant usage
  - _Requirements: 6.4_

- [ ] 8. Implement comprehensive validation integration
  - [ ] 8.1 Add machine-level network configuration validation
    - Create `validateNetworkConfiguration()` method in machine service
    - Implement pre-deployment network validation checks
    - Add comprehensive error reporting for network validation failures
    - Write unit tests for service-level network validation
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ] 8.2 Integrate validation with deployment workflow
    - Add network validation step to machine deployment workflow
    - Implement proper error handling and status updates for validation failures
    - Add condition updates for network validation results
    - Write integration tests for validation workflow
    - _Requirements: 4.4, 6.4_

- [ ] 9. Create comprehensive test suite for static IP functionality
  - [ ] 9.1 Write API validation tests
    - Create tests for static IP configuration acceptance and rejection
    - Test IP format validation edge cases
    - Verify conflict detection across multiple machines
    - Test backward compatibility with existing DHCP configurations
    - _Requirements: 3.3, 4.1, 4.2_

  - [ ] 9.2 Write integration tests for end-to-end workflows
    - Create tests for complete static IP assignment workflow
    - Test mixed environments with DHCP and static IP VMs
    - Verify status reporting accuracy for network configuration
    - Test error recovery and retry scenarios
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 10. Update provider ID parsing for network-aware VMs
  - Ensure existing provider ID parsing works with static IP VMs
  - Add any necessary network information to provider ID metadata
  - Verify provider ID uniqueness with network configuration
  - Write unit tests for provider ID handling with static IPs
  - _Requirements: 6.1_