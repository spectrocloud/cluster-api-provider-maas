# LXD Integration Testing Guide

This guide provides instructions for testing the LXD integration in the CAPMAAS controller.

## Prerequisites

1. A running MAAS server with API access
2. A Kubernetes cluster with Cluster API installed
3. The newly built CAPMAAS controller image:
   ```
   gcr.io/spectro-dev-public/release/cluster-api/cluster-api-provider-maas-controller-amd64:v0.6.0-spectro-4.0.0-dev-10052025
   ```

## Setup

### 1. Create MAAS Credentials Secret

Create a Kubernetes secret containing your MAAS credentials:

```bash
kubectl create namespace capmaas-system
kubectl create secret generic maas-credentials \
  --namespace capmaas-system \
  --from-literal=url=http://your-maas-server:5240/MAAS \
  --from-literal=token=your-maas-api-token
```

### 2. Deploy CAPMAAS Controller

Create a file named `capmaas-deployment.yaml` with the following content:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: capmaas-controller-manager
  namespace: capmaas-system
  labels:
    control-plane: capmaas-controller-manager
spec:
  selector:
    matchLabels:
      control-plane: capmaas-controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: capmaas-controller-manager
    spec:
      containers:
      - name: manager
        image: gcr.io/spectro-dev-public/release/cluster-api/cluster-api-provider-maas-controller-amd64:v0.6.0-spectro-4.0.0-dev-10052025
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        env:
        - name: MAAS_API_URL
          valueFrom:
            secretKeyRef:
              name: maas-credentials
              key: url
              optional: true
        - name: MAAS_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: maas-credentials
              key: token
              optional: true
```

Apply the deployment:

```bash
kubectl apply -f capmaas-deployment.yaml
```

### 3. Create an Infrastructure Cluster with LXD Support

Create a file named `infrastructure-cluster.yaml` with the following content:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: infra-cluster
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks: ["192.168.0.0/16"]
    serviceDomain: "cluster.local"
    services:
      cidrBlocks: ["10.96.0.0/12"]
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: MaasCluster
    name: infra-cluster
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: MaasCluster
metadata:
  name: infra-cluster
  namespace: default
spec:
  dnsDomain: "maas"
  lxdControlPlaneCluster: true
  lxdConfig:
    enabled: true
    storageBackend: "zfs"
    storageSize: "50"
    networkBridge: "br0"
    resourcePool: "default"
    skipNetworkUpdate: true
```

Apply the cluster configuration:

```bash
kubectl apply -f infrastructure-cluster.yaml
```

### 4. Create Control Plane Machines

Create a file named `control-plane.yaml` with the following content:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Machine
metadata:
  name: infra-cluster-cp-0
  namespace: default
  labels:
    cluster.x-k8s.io/cluster-name: infra-cluster
    cluster.x-k8s.io/control-plane: "true"
spec:
  clusterName: infra-cluster
  bootstrap:
    dataSecretName: "bootstrap-data"  # This would normally be created by a bootstrap provider
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: MaasMachine
    name: infra-cluster-cp-0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: MaasMachine
metadata:
  name: infra-cluster-cp-0
  namespace: default
  labels:
    cluster.x-k8s.io/cluster-name: infra-cluster
    cluster.x-k8s.io/control-plane: "true"
spec:
  minCPU: 2
  minMemory: 4096
  image: "ubuntu/focal"
```

Apply the control plane configuration:

```bash
kubectl apply -f control-plane.yaml
```

## Monitoring and Verification

### 1. Monitor the CAPMAAS Controller Logs

```bash
kubectl logs -f -n capmaas-system deployment/capmaas-controller-manager
```

### 2. Check the Status of the MaasCluster

```bash
kubectl get maascluster infra-cluster -o yaml
```

Look for the `LXDReadyCondition` in the status section.

### 3. Check MAAS for LXD Host Registration

1. Log in to your MAAS web interface
2. Go to the "KVM" section
3. Verify that your control plane machine has been registered as an LXD host

## Testing Workload Clusters

Once your infrastructure cluster is ready, you can create a workload cluster that uses LXD VMs:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: workload-cluster
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks: ["192.168.0.0/16"]
    serviceDomain: "cluster.local"
    services:
      cidrBlocks: ["10.96.0.0/12"]
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: MaasCluster
    name: workload-cluster
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: MaasCluster
metadata:
  name: workload-cluster
  namespace: default
spec:
  dnsDomain: "maas"
  infrastructureClusterRef:
    name: infra-cluster
  workloadClusterConfig:
    controlPlanePool:
      name: "cp-pool"
      useLXD: true
```

## Troubleshooting

### Common Issues

1. **MAAS Credentials Not Found**
   - Check that the `maas-credentials` secret exists and contains the correct keys
   - Verify that the environment variables are set correctly

2. **LXD Host Registration Fails**
   - Check that the control plane machine has been provisioned correctly
   - Verify that the machine has network connectivity to the MAAS server
   - Check that the LXD service is running on the machine

3. **VM Creation Fails**
   - Check that the LXD host has been registered correctly
   - Verify that the LXD host has enough resources to create VMs
   - Check that the storage pool and network bridge exist on the LXD host

## Cleanup

To clean up the resources:

```bash
kubectl delete -f control-plane.yaml
kubectl delete -f infrastructure-cluster.yaml
kubectl delete -f capmaas-deployment.yaml
kubectl delete secret -n capmaas-system maas-credentials
kubectl delete namespace capmaas-system
``` 