# Cluster-API-Provider-MAAS
Cluster API Provider for Canonical Metal-As-A-Service [maas.io](https://maas.io/)

You're welcome to join the upcoming [webinar](https://www.spectrocloud.com/webinars/managing-bare-metal-k8s-like-any-other-cluster/) for capmaas!


# Getting Started

## Public Images
Spectro Cloud public images

| Kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.25.6             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1256-0.tar.gz       |
| 1.26.1             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1261-0.tar.gz       |



## Custom Image Generation
Refer [image-generation/](image-generation/README.md)

## MAAS In-Memory Deployment

MAAS in-memory deployment allows machines to run directly from RAM without writing to disk. This feature is useful for ephemeral workloads, testing environments, or scenarios where you want to preserve disk state.

### MAAS Version Compatibility

To use the in-memory deployment feature, your MAAS installation must be running one of the following versions:

| MAAS Version | Status      |
|--------------|-------------|
| \>= 3.5.10   | ✅ Supported |
| \>= 3.6.3    | ✅ Supported |
| \>= 3.7.1    | ✅ Supported |

### Requirements

- **Minimum RAM**: Machines must have at least **16GB of RAM** for proper functionality when using in-memory deployment.

### Usage Example

To enable in-memory deployment, set `deployInMemory: true` in your `MaasMachineTemplate` spec:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: MaasMachineTemplate
metadata:
  annotations:
  name: mt-worker-memory
  namespace: test
spec:
  template:
    spec:
      deployInMemory: true
      image: custom/your-image
      minCPU: 4
      minMemory: 16384
      resourcePool: default
      tags:
      - memory
```

## Hello world

- Create kind cluster
```bash
kind create cluster --name=maas-cluster
```

- Install clusterctl v1beta1
https://release-1-1.cluster-api.sigs.k8s.io/user/quick-start.html

- Setup clusterctl configuration `~/.cluster-api/clusterctl.yaml`
```
# MAAS access endpoint and key
MAAS_API_KEY: <maas-api-key>
MAAS_ENDPOINT: http://<maas-endpoint>/MAAS
MAAS_DNS_DOMAIN: maas.domain

# Cluster configuration
KUBERNETES_VERSION: v1.26.4
CONTROL_PLANE_MACHINE_IMAGE: custom/u-2204-0-k-1264-0
CONTROL_PLANE_MACHINE_MINCPU: 4
CONTROL_PLANE_MACHINE_MINMEMORY: 8192
WORKER_MACHINE_IMAGE: custom/u-2204-0-k-1264-0
WORKER_MACHINE_MINCPU: 4
WORKER_MACHINE_MINMEMORY: 8192

# Selecting machine based on resourcepool (optional) and machine tag (optional)
CONTROL_PLANE_MACHINE_RESOURCEPOOL: resorcepool-controller
CONTROL_PLANE_MACHINE_TAG: hello-world
WORKER_MACHINE_RESOURCEPOOL: resourcepool-worker
WORKER_MACHINE_TAG: hello-world
```
- Initialize infrastructure
```bash
clusterctl init --infrastructure maas:v0.7.0
```
- Generate and create cluster
```
clusterctl generate cluster t-cluster --infrastructure=maas:v0.7.0 --kubernetes-version v1.26.4 --control-plane-machine-count=1 --worker-machine-count=3 | kubectl apply -f -
```

## Developer Guide
- Create kind cluster
```shell
kind create cluster --name=maas-cluster
```

- Install clusterctl v1 depending on the version you are working with

- Makefile set IMG=<your docker repo>
- Run 
```shell
make docker-build && make docker-push
```
    
- Generate dev manifests
```shell
make dev-manifests
```

- Move _build/dev/ directory contents to ~/.clusterapi/overrides v0.5.0 depending on version you are working with

```text
.
├── clusterctl.yaml
├── overrides
│   ├── infrastructure-maas
│       └── v0.5.0
│           ├── cluster-template.yaml
│           ├── infrastructure-components.yaml
│           └── metadata.yaml
└── version.yaml

```

- Run
```shell
clusterctl init --infrastructure maas:v0.7.0
```


## Install CRDs
### v1beta1 v0.7.0 release
- Generate cluster using
```shell
clusterctl generate cluster t-cluster  --infrastructure=maas:v0.7.0 | kubectl apply -f -
```
or
```shell
clusterctl generate cluster t-cluster --infrastructure=maas:v0.7.0 --kubernetes-version v1.26.4 > my_cluster.yaml
kubectl apply -f my_cluster.yaml
```
or
```shell
clusterctl generate cluster t-cluster --infrastructure=maas:v0.7.0 --kubernetes-version v1.26.4 --control-plane-machine-count=1 --worker-machine-count=3 > my_cluster.yaml
kubectl apply -f my_cluster.yaml
```