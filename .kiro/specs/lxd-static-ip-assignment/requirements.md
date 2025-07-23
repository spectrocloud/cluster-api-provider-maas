# Requirements Document

## Introduction

This feature extends the existing LXD VM provisioning capability to support static IP assignment for LXD virtual machines. Currently, LXD VMs rely on DHCP for IP address assignment, but this enhancement will allow users to configure specific static IP addresses for their LXD VMs, providing more predictable networking and better integration with existing infrastructure.

## Requirements

### Requirement 1

**User Story:** As a cluster administrator, I want to assign static IP addresses to LXD VMs, so that I can have predictable network addressing for my infrastructure components.

#### Acceptance Criteria

1. WHEN a user configures an LXD VM with a static IP address THEN the system SHALL validate that the IP address is in a valid format
2. WHEN a user configures an LXD VM with a static IP address THEN the system SHALL validate that the IP address is within the allowed network range
3. WHEN an LXD VM is provisioned with a static IP configuration THEN the system SHALL configure the VM's network interface with the specified static IP
4. IF a static IP address is already in use THEN the system SHALL reject the configuration and return an appropriate error message

### Requirement 2

**User Story:** As a cluster administrator, I want to configure network settings including gateway and DNS servers for static IP assignments, so that LXD VMs have complete network connectivity.

#### Acceptance Criteria

1. WHEN a static IP is configured THEN the system SHALL allow specification of a gateway IP address
2. WHEN a static IP is configured THEN the system SHALL allow specification of one or more DNS server addresses
3. WHEN a static IP is configured THEN the system SHALL allow specification of a subnet mask or CIDR notation
4. IF gateway or DNS servers are not specified THEN the system SHALL use reasonable defaults from the network configuration

### Requirement 3

**User Story:** As a cluster administrator, I want static IP configuration to be optional, so that existing DHCP-based workflows continue to work without changes.

#### Acceptance Criteria

1. WHEN no static IP configuration is provided THEN the system SHALL default to DHCP-based IP assignment
2. WHEN static IP configuration is provided THEN the system SHALL use static IP assignment instead of DHCP
3. WHEN updating an existing LXD VM configuration THEN the system SHALL allow switching between static and DHCP modes

### Requirement 4

**User Story:** As a cluster administrator, I want to validate static IP assignments before VM creation, so that I can catch configuration errors early in the deployment process.

#### Acceptance Criteria

1. WHEN static IP configuration is submitted THEN the system SHALL validate IP address format using standard validation
2. WHEN static IP configuration is submitted THEN the system SHALL check for IP address conflicts within the cluster
3. WHEN static IP configuration is submitted THEN the system SHALL validate that the gateway is reachable from the specified IP
4. IF validation fails THEN the system SHALL prevent VM creation and provide clear error messages

### Requirement 5

**User Story:** As a cluster administrator, I want to specify network interface configuration for static IP assignment, so that I can control which network interface receives the static IP.

#### Acceptance Criteria

1. WHEN configuring static IP THEN the system SHALL allow specification of the target network interface name
2. WHEN no interface is specified THEN the system SHALL use the default network interface for the LXD VM
3. WHEN multiple network interfaces are present THEN the system SHALL apply static IP configuration only to the specified interface

### Requirement 6

**User Story:** As a developer, I want the static IP configuration to integrate seamlessly with the existing MaasMachine API, so that the feature follows established patterns and is easy to use.

#### Acceptance Criteria

1. WHEN static IP configuration is added to MaasMachine spec THEN the system SHALL maintain backward compatibility with existing configurations
2. WHEN static IP configuration is present THEN the system SHALL include the configuration in the MaasMachine status
3. WHEN LXD VM provisioning fails due to network configuration THEN the system SHALL update the MaasMachine status with appropriate error conditions
4. WHEN static IP is successfully assigned THEN the system SHALL reflect the assigned IP in the MaasMachine status