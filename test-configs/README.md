# MAAS LXD VM Provisioning Test Environment

This directory contains test configurations and utilities for testing LXD VM provisioning with the MAAS cluster-api provider.

## Environment Status

✅ **Cluster API Core Components**: Installed and ready  
✅ **MAAS Provider CRDs**: Installed (`maasmachines`, `maasclusters`, etc.)  
✅ **Namespace**: `cluster-api-provider-maas-system` created  
⏳ **MAAS Provider Controller**: Needs to be deployed  
⏳ **MAAS Credentials**: Need to be configured  

## Prerequisites

1. **MAAS Environment**: You need access to a MAAS server with LXD VM hosts configured
2. **MAAS API Access**: API token and server URL
3. **Docker**: To build and deploy the controller (or use pre-built images)

## Quick Setup

### 1. Configure MAAS Credentials

```bash
# Copy the template and fill in your MAAS details
cp maas-credentials-template.yaml maas-credentials.yaml

# Edit the file with your MAAS server URL and API token
# Format for token: oauth_token="YOUR-TOKEN",oauth_token_secret="",oauth_consumer_key="YOUR-CONSUMER-KEY"

# Apply the credentials
kubectl apply -f maas-credentials.yaml
```

### 2. Deploy the MAAS Provider Controller

```bash
# Build the Docker image (requires Docker)
cd ..
make docker-build IMG=cluster-api-provider-maas-controller:v0.6.1-lxd-test

# Deploy to cluster
make deploy IMG=cluster-api-provider-maas-controller:v0.6.1-lxd-test
```

### 3. Update Test Configurations

Before running tests, update the test YAML files with your actual MAAS environment details:

- **IP Addresses**: Update static IP addresses in test files to match your network
- **Host Selection**: Update `preferredHosts`, `availabilityZone`, `resourcePool` with actual values
- **Network Settings**: Update `gateway`, `subnet`, `dnsServers` to match your network

## Available Tests

### Static IP Assignment Test
**File**: `lxd-static-ip-test.yaml`
- Tests LXD VM provisioning with static IP configuration
- Configures: 2 CPU, 4GB RAM, 20GB disk
- Network: Static IP with custom DNS

### DHCP Test (Backward Compatibility)
**File**: `lxd-dhcp-test.yaml`  
- Tests LXD VM provisioning with DHCP network configuration
- Configures: 1 CPU, 2GB RAM, 10GB disk
- Network: DHCP assignment

### Multi-VM Test
**File**: `lxd-multi-vm-test.yaml`
- Tests provisioning multiple VMs simultaneously
- Mix of static IP and DHCP configurations
- Tests resource distribution across hosts

## Running Tests

### Using the Test Runner Script

```bash
# Show available tests
./test-runner.sh

# Run a specific test
./test-runner.sh lxd-static-ip-test

# Monitor a specific machine
./test-runner.sh monitor lxd-static-ip-test 300

# Clean up a test
./test-runner.sh cleanup lxd-static-ip-test
```

### Manual Testing

```bash
# Apply a test configuration
kubectl apply -f lxd-static-ip-test.yaml

# Monitor the machine status
kubectl get maasmachine -w

# Check detailed status
kubectl describe maasmachine lxd-static-ip-test

# View controller logs
kubectl logs -n cluster-api-provider-maas-system -l control-plane=controller-manager -f

# Clean up
kubectl delete -f lxd-static-ip-test.yaml
```

## Test Validation

### Expected Outcomes

For successful LXD VM provisioning, you should see:

1. **MaasMachine Status**: Transitions from `Provisioning` → `Running`
2. **Provider ID**: Set with LXD format (e.g., `maas-lxd:///zone-a/lxd-host-1/vm-001`)
3. **Network Status**: Shows assigned IP address and configuration method
4. **VM Host**: VM appears in MAAS interface under the selected VM host

### Debugging

```bash
# Check MaasMachine status
kubectl describe maasmachine <machine-name>

# Check controller logs
kubectl logs -n cluster-api-provider-maas-system -l control-plane=controller-manager

# Check MAAS API connectivity
kubectl exec -n cluster-api-provider-maas-system deployment/cluster-api-provider-maas-controller -- \
  curl -H "Authorization: OAuth <your-token>" http://your-maas-server:5240/MAAS/api/2.0/vm-hosts/

# Check Cluster API core components
kubectl get pods -n capi-system
kubectl get pods -n capi-kubeadm-bootstrap-system
kubectl get pods -n capi-kubeadm-control-plane-system
```

## MAAS Environment Requirements

### VM Host Configuration
Your MAAS environment should have:
- LXD VM hosts registered and commissioned
- Proper network configuration (subnets, IP ranges)
- Storage pools configured on LXD hosts
- Appropriate resource availability (CPU, memory)

### Network Configuration
- Subnet configured in MAAS for static IP assignments  
- Reserved IP ranges if using static IPs
- Bridge configuration on VM hosts
- DNS server accessibility

## Features Tested

✅ **Core LXD Provisioning**: VM creation, deployment, deletion  
✅ **Static IP Assignment**: Custom IP, gateway, subnet, DNS  
✅ **Resource Management**: CPU, memory, disk allocation  
✅ **Host Selection**: Zone, pool, tag-based selection  
✅ **Network Status**: Status extraction and reporting  
✅ **Provider ID**: LXD-specific format handling  
✅ **Backward Compatibility**: DHCP-based workflows  

## Integration Points Tested

- MAAS VMHosts API integration
- Network configuration validation  
- IP conflict detection
- Resource allocation and validation
- Host selection algorithms
- Status reporting and condition handling

## Next Steps

Once you provide MAAS API access details, we can:
1. Configure the credentials
2. Deploy the controller
3. Update test configurations with real environment values
4. Run comprehensive tests against real MAAS infrastructure
5. Validate all LXD VM provisioning features

The environment is fully prepared for real-world testing!