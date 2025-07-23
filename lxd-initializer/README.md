# LXD Initializer DaemonSet

This directory contains the code and configuration for the LXD initializer DaemonSet, which is responsible for initializing LXD on each node and registering it with MAAS as a VM host.

## Architecture

The LXD initializer uses a Kubernetes DaemonSet to ensure that LXD is properly initialized on each node. This approach has several advantages:

1. **No direct access required**: The CAPMaaS controller doesn't need direct access to the LXD socket on each node
2. **Proper initialization**: LXD is initialized locally on each node by the DaemonSet
3. **Automatic registration**: Each node is automatically registered with MAAS as a VM host
4. **Scalability**: Works correctly in a distributed environment

## Components

1. **lxd-initializer.go**: Go program that initializes LXD and registers the node with MAAS
2. **Dockerfile**: Dockerfile for building the LXD initializer container
3. **lxd-initializer-daemonset.yaml**: Kubernetes DaemonSet manifest
4. **Makefile**: Makefile for building and pushing the LXD initializer
5. **go.mod**: Go module file for the LXD initializer

## Building and Deploying

1. Build the LXD initializer:

```bash
make build
```

2. Build and push the Docker image:

```bash
make docker-push REGISTRY=<your-registry> TAG=<your-tag>
```

3. Update the DaemonSet manifest with your registry and tag:

```bash
sed -i 's/${REGISTRY}/<your-registry>/g' lxd-initializer-daemonset.yaml
sed -i 's/${TAG}/<your-tag>/g' lxd-initializer-daemonset.yaml
```

4. Apply the DaemonSet manifest:

```bash
kubectl apply -f lxd-initializer-daemonset.yaml
```

## Configuration

The LXD initializer supports the following environment variables:

- `NODE_NAME`: Name of the node (required)
- `STORAGE_BACKEND`: Storage backend to use (default: "zfs")
- `STORAGE_SIZE`: Storage size in GB (default: "50")
- `NETWORK_BRIDGE`: Network bridge to use (default: "br0")
- `MAAS_API_KEY`: MAAS API key for VM host registration
- `MAAS_ENDPOINT`: MAAS API endpoint for VM host registration
- `ZONE`: Zone for the VM host (default: "default")
- `RESOURCE_POOL`: Resource pool for the VM host (default: "default")

## Integration with CAPMaaS

The LXD initializer is designed to work with the CAPMaaS controller. When a new node is added to the cluster, the DaemonSet will automatically initialize LXD on the node and register it with MAAS as a VM host.

The CAPMaaS controller can then use the MAAS API to create and manage VMs on the registered VM hosts, without needing direct access to the LXD socket on each node. 