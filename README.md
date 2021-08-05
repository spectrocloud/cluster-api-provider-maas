# cluster-api-provider-maas
Cluster API Provider MaaS


# Getting Started

## Image Generation


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

    
    

