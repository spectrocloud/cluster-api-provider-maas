# cluster-api-provider-maas
Cluster API Provider for Canonical Metal-As-A-Service [maas.io](https://maas.io/)

Welcome to join the upcoming [webinar](https://www.spectrocloud.com/webinars/managing-bare-metal-k8s-like-any-other-cluster/) for capmaas!


# Getting Started

## Public Images
spectrocloud public images

| kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.18.19            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.18.19.tar.gz |
| 1.19.13            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.19.13.tar.gz |
| 1.20.9             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.9.tar.gz  |
| 1.21.2             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.21.2.tar.gz  |



## Custom Image Generation
Refer [image-generation/](image-generation/README.md)

## Set up

### v1beta1
create kind cluster

```bash
kind create cluster
```

install clusterctl v1beta1
https://release-1-1.cluster-api.sigs.k8s.io/user/quick-start.html

run
```bash
clusterctl init --infrastructure maas:v0.3.0
```



### v1alph4
create kind cluster

```bash
kind create cluster
```

install clusterctl v1alpha4
https://release-0-4.cluster-api.sigs.k8s.io/user/quick-start.html

run
```bash
clusterctl init --infrastructure maas:v0.2.0
```


### v1alpha3
    
create kind cluster
    
```shell
kind create cluster
```

install clusterctl v1alpha3
    https://release-0-3.cluster-api.sigs.k8s.io/user/quick-start.html

run
```shell
clusterctl init --infrastructure maas:v0.1.1
```


### Developer Guide
create kind cluster

```shell
kind create cluster
```

install clusterctl v3/v4 depending on the version you are working with

Makefile set IMG=<your docker repo>
run 
```shell
make docker-build && make docker-push
```
    
generate dev manifests
```shell
make dev-manifests
```

move 
    _build/dev/

directory contents to ~/.clusterapi/overrides v0.1.0 or v0.2.0 or v0.3.0 depending on version you are working with

```text
.
├── clusterctl.yaml
├── overrides
│   ├── infrastructure-maas
│       ├── v0.1.1
│       │   ├── cluster-template.yaml
│       │   ├── infrastructure-components.yaml
│       │   └── metadata.yaml
│       ├── v0.2.0
│       │   ├── cluster-template.yaml
│       │   ├── infrastructure-components.yaml
│       │   └── metadata.yaml
│       └── v0.3.0
│           ├── cluster-template.yaml
│           ├── infrastructure-components.yaml
│           └── metadata.yaml
└── version.yaml

```


run
```shell
clusterctl init --infrastructure maas:v0.1.1
or 
clusterctl init --infrastructure maas:v0.2.0
or
clusterctl init --infrastructure maas:v0.3.0
```


## install CRDs 

### v1alpha3 v0.1.1 release
run example from for v1alpha3 or v0.1.0 release
```shell
kubectl apply -f examples/sample-with-workerpool.yaml
```

### v1alpah4 v0.2.0 release
generate cluster using
```shell
clusterctl generate cluster t-cluster  --infrastructure=maas:v0.2.0
```

### v1beta1 v0.3.0 release
generate cluster using
```shell
clusterctl generate cluster t-cluster  --infrastructure=maas:v0.3.0
```
