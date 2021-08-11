# cluster-api-provider-maas
Cluster API Provider MaaS


# Getting Started

## Image Generation
Refer [image-generation/](image-generation/README.md)

## Set up
    
create kind cluster
    
```bash
kind create cluster
```

install clusterctl v3
    https://release-0-3.cluster-api.sigs.k8s.io/user/quick-start.html

run
```
clusterctl init --core cluster-api:v0.3.19 --bootstrap  kubeadm:v0.3.19 --control-plane  kubeadm:v0.3.19
```

Makefile set IMG=<your docker repo>
run 
```
make docker-build && make docker-push
```
    
generate dev manifests
```
make dev-manifests
```

edit _build/dev/infrastructure-components.yaml
```yaml
apiVersion: v1
kind: Secret
metadata:
  labels:
    cluster.x-k8s.io/provider: infrastructure-maas
  name: capmaas-manager-bootstrap-credentials
  namespace: capmaas-system
stringData:
  MAAS_API_KEY: _ #${MAAS_API_KEY}
  MAAS_ENDPOINT: _ #${MAAS_ENDPOINT}
type: Opaque
```

run
```shell
kubectl apply -f _build/dev/infrastructure-components.yaml
```

wait for capi and capmaas pods to be running

