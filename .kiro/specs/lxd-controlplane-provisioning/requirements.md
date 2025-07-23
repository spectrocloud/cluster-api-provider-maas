# Requirements Document

## Introduction

This feature enhances the cluster-api-provider-maas v4.6.0-spectro to support dynamic creation of LXD virtual machines inside available LXD hosts specifically for control plane nodes. The primary goal is to improve resource efficiency by allowing smaller-sized control plane instances on large bare metal machines, enabling multiple LXD VMs per host while maintaining high availability through resource pool management and availability zone distribution. This enhancement integrates with the existing Spectro-specific features including custom endpoints, preferred subnet support, and enhanced namespace management.

## Requirements

### Requirement 1

**User Story:** As a cluster administrator, I want to optionally provision control plane nodes as LXD VMs instead of bare metal machines, so that I can improve resource utilization on large bare metal hosts.

#### Acceptance Criteria

1. WHEN creating a control plane node THEN the system SHALL provide an option to use LXD VM provisioning via a `provisioningMode` field
2. WHEN `provisioningMode` is set to "lxd" THEN the system SHALL create a virtual machine on an available LXD host
3. WHEN `provisioningMode` is set to "bare-metal" or omitted THEN the system SHALL continue to use bare metal provisioning as before
4. WHEN provisioning LXD VMs THEN the system SHALL allow configuration of VM resource specifications (CPU, memory, disk) through existing `minCPU` and `minMemory` fields

### Requirement 2

**User Story:** As a cluster administrator, I want to specify a resource pool for LXD hosts, so that I can dedicate specific hosts for control plane VM provisioning and maintain resource isolation.

#### Acceptance Criteria

1. WHEN configuring LXD VM provisioning THEN the system SHALL use the existing `resourcePool` field to target LXD-capable hosts
2. WHEN a resource pool is specified THEN the system SHALL only provision LXD VMs on hosts within that pool that support LXD
3. WHEN no resource pool is specified THEN the system SHALL use any available LXD-capable host
4. IF the specified resource pool contains no available LXD hosts THEN the system SHALL return an error with clear messaging

### Requirement 3

**User Story:** As a cluster administrator, I want LXD VMs to be distributed across multiple availability zones when the resource pool spans multiple AZs, so that I can maintain high availability for my control plane.

#### Acceptance Criteria

1. WHEN the target resource pool spans multiple availability zones THEN the system SHALL distribute LXD VMs across different AZs using existing `failureDomain` support
2. WHEN provisioning multiple control plane VMs THEN the system SHALL attempt to place them in different availability zones
3. WHEN an availability zone becomes unavailable THEN the remaining VMs in other AZs SHALL continue to function
4. IF insufficient availability zones are available for the requested number of control plane nodes THEN the system SHALL distribute VMs as evenly as possible across available AZs

### Requirement 4

**User Story:** As a cluster administrator, I want the system to automatically select appropriate LXD hosts based on available resources, so that VM provisioning succeeds without manual host selection.

#### Acceptance Criteria

1. WHEN provisioning a LXD VM THEN the system SHALL check available resources (CPU, memory, disk) on candidate LXD hosts
2. WHEN multiple suitable hosts are available THEN the system SHALL select the most appropriate host based on resource availability
3. WHEN no suitable host has sufficient resources THEN the system SHALL return an error indicating resource constraints
4. WHEN a selected LXD host becomes unavailable during provisioning THEN the system SHALL retry with another suitable host

### Requirement 5

**User Story:** As a developer integrating with cluster-api-provider-maas, I want the LXD VM provisioning to be transparent to existing cluster-api workflows, so that existing automation and tooling continue to work.

#### Acceptance Criteria

1. WHEN using LXD VM provisioning THEN the cluster-api Machine resource interface SHALL remain unchanged
2. WHEN a LXD VM is provisioned THEN it SHALL appear as a normal Machine resource in the cluster
3. WHEN querying Machine status THEN LXD VMs SHALL report status information consistent with bare metal machines
4. WHEN deleting a Machine backed by LXD VM THEN the system SHALL properly clean up both the VM and any associated resources

### Requirement 6

**User Story:** As a cluster administrator using Spectro-specific features, I want LXD VM provisioning to work seamlessly with custom endpoints and preferred subnets, so that my existing configurations remain functional.

#### Acceptance Criteria

1. WHEN using custom endpoint annotations with LXD VMs THEN the system SHALL skip DNS reconciliation as with bare metal machines
2. WHEN preferred subnet configuration is specified THEN LXD VMs SHALL select IP addresses from preferred subnets for DNS attachment
3. WHEN namespace-scoped controllers are used THEN LXD VM provisioning SHALL respect namespace boundaries
4. WHEN using custom ports for API server endpoints THEN LXD VMs SHALL properly register with the specified port

### Requirement 7

**User Story:** As a cluster administrator, I want proper error handling and logging for LXD VM operations, so that I can troubleshoot issues and monitor the provisioning process.

#### Acceptance Criteria

1. WHEN LXD VM provisioning fails THEN the system SHALL log detailed error information including host selection and resource constraints
2. WHEN LXD host communication fails THEN the system SHALL retry with exponential backoff and log retry attempts
3. WHEN LXD VM creation succeeds THEN the system SHALL log VM details including host location and resource allocation
4. WHEN LXD VM deletion occurs THEN the system SHALL log cleanup operations and verify successful resource deallocation

### Requirement 8

**User Story:** As a system integrator, I want LXD VM provisioning to integrate with existing MAAS client patterns and error handling, so that the implementation follows established conventions.

#### Acceptance Criteria

1. WHEN implementing LXD VM support THEN the system SHALL use the existing `github.com/spectrocloud/maas-client-go` library patterns
2. WHEN LXD operations fail THEN the system SHALL use existing error handling and condition reporting mechanisms
3. WHEN LXD VMs are provisioned THEN the system SHALL follow existing reconciliation patterns with appropriate requeue intervals
4. WHEN generating provider IDs for LXD VMs THEN the system SHALL maintain the existing format while distinguishing VM resources