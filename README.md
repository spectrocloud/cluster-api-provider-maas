# Cluster-API-Provider-MAAS
Cluster API Provider for Canonical Metal-As-A-Service [maas.io](https://maas.io/)

Welcome to join the upcoming [webinar](https://www.spectrocloud.com/webinars/managing-bare-metal-k8s-like-any-other-cluster/) for capmaas!


# Getting Started

## Public Images
Spectro Cloud public images

| Kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.21.14            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12114-0.tar.gz      |
| 1.22.12            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12212-0.tar.gz      |
| 1.23.9             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1239-0.tar.gz       |
| 1.24.3             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1243-0.tar.gz       |
| 1.25.6             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1256-0.tar.gz       |
| 1.26.1             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1261-0.tar.gz       |



## Custom Image Generation
Refer [image-generation/](image-generation/README.md)

## Set up

- Create kind cluster
```bash
kind create cluster
```

- Install clusterctl v1beta1
https://release-1-1.cluster-api.sigs.k8s.io/user/quick-start.html

- Run
```bash
clusterctl init --infrastructure maas:v0.4.0
```


### Developer Guide
- Create kind cluster
```shell
kind create cluster
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

- Move _build/dev/ directory contents to ~/.clusterapi/overrides v0.4.0 depending on version you are working with

```text
.
├── clusterctl.yaml
├── overrides
│   ├── infrastructure-maas
│       └── v0.4.0
│           ├── cluster-template.yaml
│           ├── infrastructure-components.yaml
│           └── metadata.yaml
└── version.yaml

```

- Run
```shell
clusterctl init --infrastructure maas:v0.4.0
```


## Install CRDs

### v1beta1 v0.4.0 release
- Generate cluster using
```shell
clusterctl generate cluster t-cluster  --infrastructure=maas:v0.4.0 | kubectl apply -f -
```
or
```shell
clusterctl generate cluster t-cluster --infrastructure=maas:v0.4.0 --kubernetes-version v1.24.3 > my_cluster.yaml
kubectl apply -f my_cluster.yaml
```
or
```shell
clusterctl generate cluster t-cluster --infrastructure=maas:v0.4.0 --kubernetes-version v1.24.3 --control-plane-machine-count=1 --worker-machine-count=3 > my_cluster.yaml
kubectl apply -f my_cluster.yaml
```