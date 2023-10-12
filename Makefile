ROOT_DIR_RELATIVE := .

include $(ROOT_DIR_RELATIVE)/common.mk

VERSION ?= $(shell cat clusterctl-settings.json | jq .config.nextVersion -r)

TOOLS_DIR := hack/tools
TOOLS_DIR_DEPS := $(TOOLS_DIR)/go.sum $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/Makefile
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
MANIFEST_DIR=_build/manifests
BUILD_DIR :=_build
RELEASE_DIR := _build/release
DEV_DIR := _build/dev
REPO_ROOT := $(shell git rev-parse --show-toplevel)
FIPS_ENABLE ?= ""
BUILDER_GOLANG_VERSION ?= 1.21
BUILD_ARGS = --build-arg CRYPTO_LIB=${FIPS_ENABLE} --build-arg BUILDER_GOLANG_VERSION=${BUILDER_GOLANG_VERSION}
ARCH ?= amd64
ALL_ARCH = amd64 arm64

RELEASE_LOC := release
ifeq ($(FIPS_ENABLE),yes)
  RELEASE_LOC := release-fips
endif

# Image URL to use all building/pushing image targets
IMAGE_NAME := cluster-api-provider-maas-controller
REGISTRY ?= gcr.io/spectro-dev-public/${RELEASE_LOC}/cluster-api
SPECTRO_VERSION ?= 4.0.0-dev
IMG_TAG ?= v0.2.0-spectro-${SPECTRO_VERSION}
CONTROLLER_IMG ?= ${REGISTRY}/${IMAGE_NAME}

# Set --output-base for conversion-gen if we are not within GOPATH
ifneq ($(abspath $(REPO_ROOT)),$(shell go env GOPATH)/src/github.com/spectrocloud/cluster-api-provider-maas)
	GEN_OUTPUT_BASE := --output-base=$(REPO_ROOT)
else
	export GOPATH := $(shell go env GOPATH)
endif

# Release images
# Release docker variables
RELEASE_REGISTRY := gcr.io/spectro-images-public/release/cluster-api-provider-maas
RELEASE_CONTROLLER_IMG := $(RELEASE_REGISTRY)/$(IMAGE_NAME)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

MOCKGEN := $(TOOLS_BIN_DIR)/mockgen
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/conversion-gen
DEFAULTER_GEN := $(TOOLS_BIN_DIR)/defaulter-gen
KUSTOMIZE := $(TOOLS_BIN_DIR)/kustomize

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests ## Run unit tests
	# TODO bring back
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet ## Build manager binary
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests ## Install CRDs into a cluster
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests ## Uninstall CRDs from a cluster
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests  ## Deploy controller in the configured Kubernetes cluster
	cd config/manager && kustomize edit set image controller=$(CONTROLLER_IMG):$(IMG_TAG)
	kustomize build config/default | kubectl apply -f -

$(MANIFEST_DIR):
	mkdir -p $(MANIFEST_DIR)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(OVERRIDES_DIR):
	@mkdir -p $(OVERRIDES_DIR)

.PHONY: dev-version-check
dev-version-check:
ifndef VERSION
	$(error VERSION must be set)
endif

.PHONY: release-version-check
release-version-check:
ifeq ($(VERSION), 0.0.0)
	$(error VERSION must be >0.0.0 for release)
endif

.PHONY: release-manifests
release-manifests: test
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(RELEASE_DIR) PULL_POLICY=IfNotPresent IMAGE=$(RELEASE_CONTROLLER_IMG):$(VERSION)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
	$(MAKE) templates OUTPUT_DIR=$(RELEASE_DIR)

.PHONY: release-overrides
release-overrides:
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(OVERRIDES_DIR) PULL_POLICY=IfNotPresent IMAGE=$(RELEASE_CONTROLLER_IMG):$(VERSION)

.PHONY: dev-manifests
dev-manifests:
	$(MAKE) manifests STAGE=dev MANIFEST_DIR=$(DEV_DIR) PULL_POLICY=Always IMAGE=$(CONTROLLER_IMG):$(IMG_TAG)
	cp metadata.yaml $(DEV_DIR)/metadata.yaml
	$(MAKE) templates OUTPUT_DIR=$(DEV_DIR)

# Generate manifests e.g. CRD, RBAC etc.
manifests: $(CONTROLLER_GEN) $(MANIFEST_DIR) $(KUSTOMIZE) $(BUILD_DIR) ## Generate manifests e.g. CRD, RBAC etc.
	rm -rf $(BUILD_DIR)/config
	cp -R config $(BUILD_DIR)/config
	sed -i'' -e 's@imagePullPolicy: .*@imagePullPolicy: '"$(PULL_POLICY)"'@' $(BUILD_DIR)/config/default/manager_pull_policy.yaml
	sed -i'' -e 's@image: .*@image: '"$(IMAGE)"'@' $(BUILD_DIR)/config/default/manager_image_patch.yaml
	"$(KUSTOMIZE)" build $(BUILD_DIR)/config/default > $(MANIFEST_DIR)/infrastructure-components.yaml

# Run go fmt against code
fmt: ## Run go fmt against code
	go fmt ./...

# Run go vet against code
vet:  ## Run go vet against code
	go vet ./...

# Generate code
generate: $(CONTROLLER_GEN) $(CONVERSION_GEN)
	$(MAKE) generate-go
	$(MAKE) generate-manifests

generate-go:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

	$(CONVERSION_GEN) \
		--input-dirs=./api/v1alpha3 \
		--extra-peer-dirs=github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha3 \
		--build-tag=ignore_autogenerated_core_v1alpha3 \
		--output-file-base=zz_generated.conversion $(GEN_OUTPUT_BASE) \
		--go-header-file=./hack/boilerplate.go.txt

.PHONY: generate-manifests
generate-manifests:  ## Generate manifests
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases


# Build the docker image
.PHONY: docker-build
docker-build: #test
	docker buildx build --load --platform linux/$(ARCH) ${BUILD_ARGS} --build-arg ARCH=$(ARCH)  --build-arg  LDFLAGS="$(LDFLAGS)" --build-arg CRYPTO_LIB=${FIPS_ENABLE} . -t $(CONTROLLER_IMG)-$(ARCH):$(IMG_TAG)

# Push the docker image
.PHONY: docker-push
docker-push: ## Push the docker image to gcr
	docker push  $(CONTROLLER_IMG)-$(ARCH):$(IMG_TAG)

## --------------------------------------
## Docker — All ARCH
## --------------------------------------
.PHONY: docker-build-all ## Build all the architecture docker images
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-push-all ## Push all the architecture docker images
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))
	$(MAKE) docker-push-manifest

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-push-manifest
docker-push-manifest: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	docker manifest create --amend $(CONTROLLER_IMG):$(IMG_TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(CONTROLLER_IMG)\-&:$(IMG_TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${CONTROLLER_IMG}:${IMG_TAG} ${CONTROLLER_IMG}-$${arch}:${IMG_TAG}; done
	docker manifest push --insecure --purge $(CONTROLLER_IMG):$(IMG_TAG)

docker-rmi: ## Remove the docker image locally
	docker rmi $(CONTROLLER_IMG):$(IMG_TAG)

mock: $(MOCKGEN)
	go generate ./...

clean-release:
	rm -rf $(RELEASE_DIR)

release: release-manifests
	$(MAKE) docker-build IMG=$(RELEASE_CONTROLLER_IMG):$(VERSION)
	$(MAKE) docker-push IMG=$(RELEASE_CONTROLLER_IMG):$(VERSION)

.PHONY: templates
templates: ## Generate release templates
	cp templates/cluster-template*.yaml $(OUTPUT_DIR)/

version: ## Prints version of current make
	@echo $(VERSION)
