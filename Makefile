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

# Image URL to use all building/pushing image targets
IMAGE_NAME := cluster-api-provider-maas
IMG_URL ?= gcr.io/$(shell gcloud config get-value project)/${USER}
IMG_TAG ?= latest
IMG ?= ${IMG_URL}/cluster-api-provider-maas:${IMG_TAG}


# Release images
# Release docker variables
RELEASE_REGISTRY := gcr.io/spectro-images-public/release
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
	cd config/manager && kustomize edit set image controller=${IMG}
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

.PHONY: release-overrides
release-overrides:
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(OVERRIDES_DIR) PULL_POLICY=IfNotPresent IMAGE=$(RELEASE_CONTROLLER_IMG):$(VERSION)

.PHONY: dev-manifests
dev-manifests:
	$(MAKE) manifests STAGE=dev MANIFEST_DIR=$(DEV_DIR) PULL_POLICY=Always IMAGE=$(IMG)
	cp metadata.yaml $(DEV_DIR)/metadata.yaml

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
generate: $(CONTROLLER_GEN)
	$(MAKE) generate-go
	$(MAKE) generate-manifests

generate-go:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

generate-manifests:  ## Generate manifests
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases


# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push: ## Push the docker image to gcr
	docker push ${IMG}

docker-rmi: ## Remove the docker image locally
	docker rmi ${IMG}

mock: $(MOCKGEN)
	go generate ./...

clean-release:
	rm -rf $(RELEASE_DIR)

release: release-manifests
	# $(MAKE) docker-build IMG=$(RELEASE_CONTROLLER_IMG):$(VERSION)
	$(MAKE) docker-push IMG=$(RELEASE_CONTROLLER_IMG):$(VERSION)

version: ## Prints version of current make
	@echo $(VERSION)
