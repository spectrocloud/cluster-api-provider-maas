# Implementation Plan

- [ ] 1. Extend MaasMachine API types with LXD configuration
  - Add ProvisioningMode enum and LXDConfig struct to MaasMachineSpec in api/v1beta1/maasmachine_types.go
  - Add LXD-specific status fields to MaasMachineStatus including LXDHost, LXDProject, and VMResourceUsage
  - Update kubebuilder validation tags and documentation for new fields
  - Generate updated CRD manifests using make manifests
  - _Requirements: 1.1, 1.4_

- [ ] 2. Create LXD data models and interfaces
  - [ ] 2.1 Define LXD provisioning service interface
    - Create pkg/maas/lxd/interfaces.go with Service interface definitions
    - Define VMSpec, LXDVMResult, LXDHost, and ResourceInfo structs
    - Add LXDCapabilities and LXDError type definitions following existing patterns
    - _Requirements: 1.1, 4.1_

  - [ ] 2.2 Implement LXD error handling types
    - Create pkg/maas/lxd/errors.go with LXDError struct and LXDErrorType enum
    - Implement error wrapping methods following existing pkg/errors patterns
    - Add error formatting and logging helpers consistent with existing error handling
    - Write unit tests for error type definitions and formatting
    - _Requirements: 7.1, 7.2, 8.2_

- [ ] 3. Implement core LXD provisioning service
  - [ ] 3.1 Create LXD host discovery and selection logic
    - Implement GetAvailableLXDHosts method in pkg/maas/lxd/service.go using existing MAAS client patterns
    - Add resource pool filtering for LXD-capable hosts following existing resourcePool logic
    - Implement SelectOptimalHost algorithm based on available CPU, memory, and disk resources
    - Add unit tests for host selection with various resource constraint scenarios
    - _Requirements: 2.1, 2.2, 4.1, 4.2_

  - [ ] 3.2 Implement availability zone distribution logic
    - Create DistributeAcrossAZs method using existing failureDomain support patterns
    - Add algorithms to evenly distribute VMs across available AZs from resource pools
    - Handle edge cases when insufficient AZs are available for requested VM count
    - Write unit tests for AZ distribution with different host and zone configurations
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [ ] 3.3 Implement VM lifecycle management methods
    - Create ComposeVM method for LXD VM creation via MAAS VMHosts API following existing allocation patterns
    - Implement DeployVM method with cloud-init user data support using existing deployment workflow
    - Add GetVM method for status checking following existing machine status patterns
    - Implement DeleteVM method with proper cleanup validation using existing release patterns
    - Write unit tests for each lifecycle operation with mock MAAS client
    - _Requirements: 1.1, 1.2, 4.3, 5.3_

- [ ] 4. Enhance machine service with LXD support
  - [ ] 4.1 Extend existing machine service interface
    - Modify DeployMachine method in pkg/maas/machine/machine.go to handle provisioning mode decisions
    - Add deployLXDVM private method following existing deployBareMetal patterns
    - Create buildVMSpec method to convert MaasMachineSpec to LXD VMSpec
    - Update existing error handling to support both bare metal and LXD error types
    - _Requirements: 1.1, 1.3, 5.1_

  - [ ] 4.2 Implement LXD VM provisioning workflow
    - Integrate LXD provisioning service with existing machine service patterns
    - Add resource validation for LXD VMs using existing minCPU and minMemory fields
    - Implement fallback to existing bare metal logic when LXD provisioning is not specified
    - Write integration tests for mixed bare metal and LXD provisioning scenarios
    - _Requirements: 1.1, 1.2, 4.4, 5.2_

- [ ] 5. Update MaasMachine controller for LXD integration
  - [ ] 5.1 Modify machine reconciliation logic
    - Update reconcileNormal method in controllers/maasmachine_controller.go to detect LXD provisioning mode
    - Add LXD-specific condition reporting using existing condition framework patterns
    - Implement status updates for LXD host information and VM resource usage
    - Ensure DNS attachment logic works with LXD VMs using existing reconcileDNSAttachment patterns
    - _Requirements: 5.1, 5.2, 5.3, 6.2_

  - [ ] 5.2 Implement LXD-specific error handling and recovery
    - Add retry logic with exponential backoff for LXD host failures using existing requeue patterns
    - Implement host selection retry when initial host becomes unavailable
    - Add comprehensive logging for LXD operations following existing logging patterns
    - Create error recovery workflows using existing condition and status update mechanisms
    - _Requirements: 4.4, 7.1, 7.2, 7.3_

  - [ ] 5.3 Update machine deletion and cleanup logic
    - Extend reconcileDelete method to handle LXD VM cleanup using existing release patterns
    - Add verification of VM resource deallocation during deletion process
    - Implement timeout handling for LXD VM deletion operations with existing requeue logic
    - Write unit tests for cleanup scenarios and error cases
    - _Requirements: 5.4, 7.4_

- [ ] 6. Add MAAS client extensions for LXD operations
  - [ ] 6.1 Research and implement MAAS VMHosts API integration
    - Investigate MAAS client library support for VMHosts operations in pkg/maas/scope/client.go
    - Add VM host discovery methods using existing client patterns and error handling
    - Implement VM composition and deployment API calls following existing machine allocation patterns
    - Create VM deletion and status checking API integrations with existing cleanup patterns
    - _Requirements: 4.1, 4.2, 8.1_

  - [ ] 6.2 Implement LXD-specific MAAS API error handling
    - Add error parsing for MAAS LXD API responses using existing error handling patterns
    - Implement retry mechanisms for transient LXD API failures with existing requeue logic
    - Add timeout handling for long-running LXD operations following existing timeout patterns
    - Write unit tests for various MAAS API error scenarios with mock client
    - _Requirements: 7.1, 7.2, 8.2_

- [ ] 7. Create comprehensive test suite for LXD functionality
  - [ ] 7.1 Write unit tests for LXD provisioning service
    - Test host selection algorithms with different resource availability scenarios
    - Test AZ distribution logic with various host configurations and zone mappings
    - Test VM lifecycle operations with mock MAAS client following existing test patterns
    - Test error handling for all defined LXD error types and recovery scenarios
    - _Requirements: 1.1, 2.1, 3.1, 4.1_

  - [ ] 7.2 Create integration tests for machine service LXD support
    - Test end-to-end LXD VM provisioning workflow with real MAAS client interactions
    - Test mixed bare metal and LXD deployments in same cluster scenarios
    - Test resource pool filtering and selection with existing resource pool configurations
    - Test controller reconciliation with LXD backend following existing controller test patterns
    - _Requirements: 1.3, 2.1, 4.1, 5.1_

  - [ ] 7.3 Implement controller tests for LXD scenarios
    - Test MaasMachine reconciliation with LXD provisioning mode using existing controller test framework
    - Test error recovery and retry mechanisms with existing error injection patterns
    - Test status updates and condition reporting following existing condition test patterns
    - Test cleanup operations for failed LXD VMs using existing cleanup test scenarios
    - _Requirements: 5.1, 5.2, 5.3, 7.1_

- [ ] 8. Add validation and defaulting for LXD configuration
  - [ ] 8.1 Implement webhook validation for LXD configuration
    - Add validation logic in api/v1beta1/maasmachine_webhook.go following existing validation patterns
    - Validate LXD configuration fields when ProvisioningMode is "lxd" using existing validation framework
    - Add resource constraint validation for LXD VMs consistent with existing minCPU/minMemory validation
    - Ensure backward compatibility with existing bare metal configurations
    - _Requirements: 1.1, 1.4_

  - [ ] 8.2 Add defaulting logic for LXD fields
    - Implement sensible defaults for LXD profile and project settings in webhook defaulting
    - Add default storage size configuration based on typical control plane requirements
    - Set ProvisioningMode default to "bare-metal" to maintain backward compatibility
    - Write unit tests for validation and defaulting scenarios using existing webhook test patterns
    - _Requirements: 1.1, 1.4_

- [ ] 9. Update provider ID generation and machine scope for LXD VMs
  - [ ] 9.1 Extend provider ID generation
    - Update GetProviderID method in pkg/maas/scope/machine.go to distinguish LXD VMs from bare metal
    - Modify provider ID format to include "maas-lxd" scheme for LXD VMs while maintaining existing format for bare metal
    - Ensure provider ID uniqueness across bare metal and LXD resources within failure domains
    - Add IsLXDProvisioning helper method to MachineScope for provisioning mode detection
    - _Requirements: 5.3, 8.4_

  - [ ] 9.2 Update existing provider ID parsing logic
    - Review and update any provider ID parsing logic to handle both "maas://" and "maas-lxd://" schemes
    - Ensure SetNodeProviderID method works correctly with LXD VM provider IDs
    - Write tests for provider ID generation and parsing with both provisioning modes
    - _Requirements: 5.3_

- [ ] 10. Integrate with existing Spectro features for LXD VMs
  - [ ] 10.1 Verify custom endpoint compatibility
    - Test that LXD VMs respect existing custom endpoint annotations using IsCustomEndpoint logic
    - Ensure DNS reconciliation is properly skipped for LXD VMs when custom endpoints are configured
    - Verify custom port configuration works with LXD VMs in existing APIServerPort logic
    - _Requirements: 6.1, 6.4_

  - [ ] 10.2 Test preferred subnet integration
    - Verify LXD VM IP addresses work with existing preferred subnet ConfigMap logic in GetPreferredSubnets
    - Test that getExternalMachineIP function works correctly with LXD VM addresses
    - Ensure DNS attachment uses preferred subnets for LXD VMs following existing patterns
    - _Requirements: 6.2_

  - [ ] 10.3 Validate namespace scoping compatibility
    - Test that LXD VM provisioning respects existing namespace boundaries in controllers
    - Verify namespace-scoped controllers work correctly with LXD provisioning
    - Ensure resource isolation is maintained with existing namespace scoping logic
    - _Requirements: 6.3_