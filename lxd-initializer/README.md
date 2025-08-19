# LXD Initializer for CAPMAAS

This component initializes LXD on nodes and registers them with MAAS as VM hosts, enabling dynamic creation of LXD VMs for Kubernetes control plane and worker nodes.

## Relationship with pkg/maas/lxd

The LXD initializer works in conjunction with the `pkg/maas/lxd` package:

- **LXD Initializer**: Runs as a DaemonSet on each node to initialize LXD and register with MAAS
- **pkg/maas/lxd**: A library used by the CAPMAAS controller to deploy the DaemonSet and interact with LXD hosts through the MAAS API

This separation allows the controller to manage LXD hosts without needing direct access to the LXD socket on each node.

## Features

- Automatic LXD initialization with configurable storage backend and network bridge
- Auto-detection of LXD socket paths for both standard and snap installations
- Skip network updates for existing networks to avoid disruption
- Automatic registration of LXD hosts with MAAS
- Auto-detection of available MAAS resource pools
- Support for running as a standalone binary or as a DaemonSet

## Usage

### Command-line Flags

The LXD initializer supports the following command-line flags:

```
--action string              Action to perform: init, register, both, or daemon (default "both")
--storage-backend string     Storage backend (dir, zfs) (default "zfs")
--storage-size string        Storage size in GB (default "50")
--network-bridge string      Network bridge name (default "br0")
--skip-network-update        Skip updating existing network (default false)
--node-ip string             Node IP address for registration
--maas-endpoint string       MAAS API endpoint
--maas-api-key string        MAAS API key
--zone string                MAAS zone for VM host (default "default")
--resource-pool string       MAAS resource pool for VM host (auto-detected if not specified)
```

### Environment Variables

The LXD initializer also supports the following environment variables:

```
NODE_NAME                    Node name
NODE_IP                      Node IP address
STORAGE_BACKEND              Storage backend (dir, zfs)
STORAGE_SIZE                 Storage size in GB
NETWORK_BRIDGE               Network bridge name
SKIP_NETWORK_UPDATE          Skip updating existing network (true/false)
MAAS_ENDPOINT                MAAS API endpoint
MAAS_API_KEY                 MAAS API key
ZONE                         MAAS zone for VM host
RESOURCE_POOL                MAAS resource pool for VM host
```

### Running Locally

To run the LXD initializer locally:

```bash
# Build the binary
make build

# Run with flags
./lxd-initializer --action=init --storage-backend=zfs --storage-size=50 --network-bridge=br0

# Or run with environment variables
STORAGE_BACKEND=zfs STORAGE_SIZE=50 NETWORK_BRIDGE=br0 ./lxd-initializer
```

### Running as a DaemonSet

To deploy the LXD initializer as a DaemonSet:

```bash
# Build and push the Docker image
make docker-build-push

# Deploy the DaemonSet
make deploy
```

## Troubleshooting

### Network Update Issues

If you encounter issues with network updates, you can skip them by setting `--skip-network-update` or `SKIP_NETWORK_UPDATE=true`.

### LXD Socket Not Found

The LXD initializer will try to find the LXD socket at the following paths:
- `/var/lib/lxd/unix.socket` (Default path)
- `/var/snap/lxd/common/lxd/unix.socket` (Snap installation path)
- `/run/lxd.socket` (Alternative path)

If your LXD socket is at a different location, you may need to modify the code.

### MAAS Registration Issues

If you encounter issues with MAAS registration, check the following:
- Ensure the MAAS API key is valid and has sufficient permissions
- Verify the MAAS endpoint is accessible from the node
- Check that the resource pool exists in MAAS (the initializer will auto-detect available pools)
- Ensure the LXD host is configured to listen on the network

## Building and Development

### Prerequisites

- Go 1.21 or later
- Docker
- Make

### Building

```bash
# Build the binary
make build

# Build the Docker image
make docker-build
```

### Testing

To test the LXD initializer:

1. Build the binary: `make build`
2. Run with the `--action=init` flag to only initialize LXD: `./lxd-initializer --action=init`
3. Run with the `--action=register` flag to only register with MAAS: `./lxd-initializer --action=register` 