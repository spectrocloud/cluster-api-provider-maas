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
ARCH ?= amd64
ALL_ARCH = amd64 arm64

## Image URL to use all building/pushing image targets
#IMAGE_NAME := cluster-api-provider-maas-controller
#IMG_URL ?= gcr.io/spectro-dev-public/release/cluster-api
#IMG_TAG ?= v0.9.0
#IMG ?= ${IMG_URL}/${IMAGE_NAME}:${IMG_TAG}

# Image URL to use all building/pushing image targets
IMAGE_NAME := cluster-api-provider-maas-controller
REGISTRY ?= "us-east1-docker.pkg.dev/spectro-images/dev/${USER}/cluster-api"
IMG_TAG ?= v0.9.0
CONTROLLER_IMG ?= ${REGISTRY}/${IMAGE_NAME}

# Set --output-base for conversion-gen if we are not within GOPATH
ifneq ($(abspath $(REPO_ROOT)),$(shell go env GOPATH)/src/github.com/spectrocloud/cluster-api-provider-maas)
	GEN_OUTPUT_BASE := --output-base=$(REPO_ROOT)
else
	export GOPATH := $(shell go env GOPATH)
endif

# Release images
# Release docker variables
RELEASE_REGISTRY := us-east1-docker.pkg.dev/spectro-public/release/cluster-api-provider-maas
RELEASE_CONTROLLER_IMG := $(RELEASE_REGISTRY)/$(IMAGE_NAME)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

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
test: generate fmt vet manifests generate-lxd-template ## Run unit tests
	# TODO bring back
	go test ./... -coverprofile cover.out

## --------------------------------------
## E2E testing (see test/e2e/README.md)
## --------------------------------------

# Image loaded into the kind management cluster; must match the mustLoad image in
# test/e2e/config/maas.yaml (the maas provider components are retagged to this via `replacements`).
E2E_CONTROLLER_IMG ?= gcr.io/spectro-images-public/release/cluster-api-provider-maas:e2e

# lxd-initializer image for e2e. Unlike the controller image (loaded into kind), this DaemonSet runs
# on the bare-metal HCP workload nodes, which pull it from a registry — so it is built AND pushed by
# `make e2e-images` and baked into the embedded DaemonSet template. Override to a registry your MAAS
# nodes can pull from (NOT the per-$(USER) dev path), e.g.
#   make test-e2e E2E_INIT_IMG=us-east1-docker.pkg.dev/spectro-images/cluster-api/lxd-initializer:e2e
E2E_INIT_IMG ?= $(INIT_RELEASE_IMG):e2e

# E2E knobs (all overridable on the command line).
E2E_CONF_FILE ?= $(REPO_ROOT)/test/e2e/config/maas.yaml
E2E_DATA_DIR ?= $(REPO_ROOT)/test/e2e/data
ARTIFACTS ?= $(REPO_ROOT)/_artifacts
GINKGO_FOCUS ?=
GINKGO_SKIP ?=
GINKGO_LABEL_FILTER ?=
GINKGO_NODES ?= 1
GINKGO_TIMEOUT ?= 3h
USE_EXISTING_CLUSTER ?= false
SKIP_RESOURCE_CLEANUP ?= false

.PHONY: generate-e2e-templates
generate-e2e-templates: $(KUSTOMIZE) ## Generate the e2e cluster-template flavor(s) from the kustomize sources
	"$(KUSTOMIZE)" build "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/main" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template.yaml"
	"$(KUSTOMIZE)" build --load_restrictor LoadRestrictionsNone "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/lxd" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template-lxd.yaml"
	"$(KUSTOMIZE)" build "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/hcp" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template-hcp.yaml"
	"$(KUSTOMIZE)" build --load_restrictor LoadRestrictionsNone "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/upgrades" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template-upgrades.yaml"
	"$(KUSTOMIZE)" build --load_restrictor LoadRestrictionsNone "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/lxd-upgrades" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template-lxd-upgrades.yaml"
	"$(KUSTOMIZE)" build "$(E2E_DATA_DIR)/infrastructure-maas/kustomize/hcp-upgrades" > "$(E2E_DATA_DIR)/infrastructure-maas/main/cluster-template-hcp-upgrades.yaml"

.PHONY: e2e-process-lxd-template
e2e-process-lxd-template: ## Bake the e2e lxd-initializer image (E2E_INIT_IMG) into the embedded DaemonSet template
	@command -v envsubst >/dev/null 2>&1 || { echo "ERROR: envsubst not found. Please install gettext package."; exit 1; }
	@echo "Processing LXD initializer template with e2e image: $(E2E_INIT_IMG)"
	LXD_INITIALIZER_IMAGE="$(E2E_INIT_IMG)" envsubst '$$LXD_INITIALIZER_IMAGE' \
		< controllers/templates/lxd_initializer_ds.yaml > controllers/templates/lxd_initializer_ds.yaml.processed
	@grep -q "$(E2E_INIT_IMG)" controllers/templates/lxd_initializer_ds.yaml.processed || { echo "ERROR: failed to bake $(E2E_INIT_IMG) into the processed template"; exit 1; }

.PHONY: e2e-lxd-initializer-image
e2e-lxd-initializer-image: ## Build and push the lxd-initializer image used by the e2e HCP workload nodes
	@echo "Building and pushing e2e lxd-initializer image: $(E2E_INIT_IMG)"
	docker buildx build --push --platform linux/$(ARCH) ${BUILD_ARGS} -f lxd-initializer/Dockerfile lxd-initializer -t $(E2E_INIT_IMG)

.PHONY: e2e-images
# Order matters: bake the initializer image into the embedded template BEFORE building the controller
# image (go:embed picks up the .processed file from the build context), and push the initializer image
# so the bare-metal HCP workload nodes can pull it.
e2e-images: e2e-process-lxd-template e2e-lxd-initializer-image ## Build the CAPMAAS controller image (+ lxd-initializer image) used by the e2e tests
	docker buildx build --load --provenance=false --sbom=false --platform linux/$(ARCH) ${BUILD_ARGS} --build-arg ARCH=$(ARCH) --build-arg LDFLAGS="$(LDFLAGS)" --build-arg CRYPTO_LIB=${FIPS_ENABLE} . -t $(E2E_CONTROLLER_IMG)
	@rm -f controllers/templates/lxd_initializer_ds.yaml.processed

.PHONY: test-e2e
test-e2e: e2e-images generate-e2e-templates ## Run the e2e tests (requires Docker, kind, and a reachable MAAS deployment)
	mkdir -p "$(ARTIFACTS)"
	go run github.com/onsi/ginkgo/v2/ginkgo -tags=e2e -v --trace \
		--nodes=$(GINKGO_NODES) --timeout=$(GINKGO_TIMEOUT) \
		--focus="$(GINKGO_FOCUS)" --skip="$(GINKGO_SKIP)" --label-filter="$(GINKGO_LABEL_FILTER)" \
		./test/e2e/... -- \
		-e2e.config="$(E2E_CONF_FILE)" \
		-e2e.artifacts-folder="$(ARTIFACTS)" \
		-e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER) \
		-e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP)

# Alias kept for parity with upstream CAPI providers.
.PHONY: e2e
e2e: test-e2e ## Alias for test-e2e

# Build manager binary
manager: generate fmt vet generate-lxd-template ## Build manager binary
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests generate-lxd-template
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
	$(MAKE) generate-lxd-template STAGE=release VERSION=$(VERSION)
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(RELEASE_DIR) PULL_POLICY=IfNotPresent IMAGE=$(RELEASE_CONTROLLER_IMG):$(VERSION) VERSION=$(VERSION)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
	$(MAKE) templates OUTPUT_DIR=$(RELEASE_DIR)

.PHONY: release-overrides
release-overrides:
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(OVERRIDES_DIR) PULL_POLICY=IfNotPresent IMAGE=$(RELEASE_CONTROLLER_IMG):$(VERSION)

.PHONY: dev-manifests
dev-manifests:
ifndef IMG
	$(error IMG is not set. Set it to the controller image you built and pushed, e.g. make dev-manifests IMG=$(CONTROLLER_IMG)-$(ARCH):$(IMG_TAG). Without it the generated infrastructure-components.yaml will have an empty controller image.)
endif
	$(MAKE) manifests STAGE=dev MANIFEST_DIR=$(DEV_DIR) PULL_POLICY=Always IMAGE=$(IMG)
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
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

	$(CONVERSION_GEN) \
		--extra-peer-dirs=github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1 \
		--output-file=zz_generated.conversion \
		--go-header-file=./hack/boilerplate.go.txt \
		./api/v1beta1

generate-manifests:  ## Generate manifests
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/...;./controllers/..." output:crd:artifacts:config=config/crd/bases


# Build the docker image
.PHONY: docker-build
docker-build: generate-lxd-template lxd-initializer-docker-build ## Build CAPMAAS controller image and the companion lxd-initializer image baked into the DaemonSet template
	@# Template is already processed by generate-lxd-template with the same image tag that lxd-initializer-docker-build builds
	docker buildx build --load --provenance=false --sbom=false --platform linux/$(ARCH) ${BUILD_ARGS} --build-arg ARCH=$(ARCH)  --build-arg  LDFLAGS="$(LDFLAGS)" --build-arg CRYPTO_LIB=${FIPS_ENABLE} . -t $(CONTROLLER_IMG)-$(ARCH):$(IMG_TAG)

# Push the docker image
.PHONY: docker-push
docker-push: lxd-initializer-docker-push ## Push the controller image and the companion lxd-initializer image to the registry
	docker push $(CONTROLLER_IMG)-$(ARCH):$(IMG_TAG)

### --------------------------------------
### Docker — All ARCH
### --------------------------------------
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
	docker rmi ${IMG}

mock: $(MOCKGEN)
	go generate ./...

clean-release:
	rm -rf $(RELEASE_DIR)

release: release-manifests
	@# Ensure template is processed with release image before building controller
	$(MAKE) generate-lxd-template STAGE=release VERSION=$(VERSION)
	@# docker-build / docker-push also build and push the companion lxd-initializer image
	$(MAKE) docker-build STAGE=release VERSION=$(VERSION)
	$(MAKE) docker-push STAGE=release VERSION=$(VERSION)

.PHONY: templates
templates: ## Generate release templates
	cp templates/cluster-template*.yaml $(OUTPUT_DIR)/

version: ## Prints version of current make
	@echo $(VERSION)

# --------------------------------------------------------------------
# LXD-initializer image (privileged DaemonSet)
# --------------------------------------------------------------------
INIT_IMAGE_NAME ?= "lxd-initializer"
INIT_IMG_TAG    ?= $(IMG_TAG)          # reuse the same tag as controller
INIT_DRI_IMG    ?= us-east1-docker.pkg.dev/spectro-images/dev/$(USER)/cluster-api/$(INIT_IMAGE_NAME)
# Release image for LXD initializer (without tag)
INIT_RELEASE_IMG ?= us-east1-docker.pkg.dev/spectro-images/cluster-api/$(INIT_IMAGE_NAME)

.PHONY: lxd-initializer-docker-build
lxd-initializer-docker-build: generate-lxd-template ## Build LXD initializer image (ensures template is processed first)
	@# Determine image to build: use INIT_DRI_IMG if explicitly set, otherwise use INIT_RELEASE_IMG for release
	@if [ -n "$(VERSION)" ] && [ "$(STAGE)" = "release" ]; then \
		BUILD_IMG="$(INIT_RELEASE_IMG):$(VERSION)"; \
	else \
		BUILD_IMG="$(INIT_DRI_IMG):$(INIT_IMG_TAG)"; \
	fi; \
	echo "Building LXD initializer image: $$BUILD_IMG"; \
	docker buildx build --load --platform linux/$(ARCH) \
	    -f lxd-initializer/Dockerfile \
	    ${BUILD_ARGS} \
	    lxd-initializer -t $$BUILD_IMG

.PHONY: lxd-initializer-docker-push
lxd-initializer-docker-push: lxd-initializer-docker-build ## Push LXD initializer image (builds first if needed)
	@# Determine image to push: use INIT_DRI_IMG if explicitly set, otherwise use INIT_RELEASE_IMG for release
	@if [ -n "$(VERSION)" ] && [ "$(STAGE)" = "release" ]; then \
		PUSH_IMG="$(INIT_RELEASE_IMG):$(VERSION)"; \
	else \
		PUSH_IMG="$(INIT_DRI_IMG):$(INIT_IMG_TAG)"; \
	fi; \
	echo "Pushing LXD initializer image: $$PUSH_IMG"; \
	docker push $$PUSH_IMG

# File target for processed template - ensures it exists before Go compilation
# This file target ensures the processed template exists, making it a proper dependency
controllers/templates/lxd_initializer_ds.yaml.processed: controllers/templates/lxd_initializer_ds.yaml
	@$(MAKE) process-lxd-initializer-template

process-lxd-initializer-template: ## Process LXD initializer template with image substitution using envsubst
	@# Check if envsubst is available
	@command -v envsubst >/dev/null 2>&1 || { echo "ERROR: envsubst not found. Please install gettext package."; exit 1; }
	@# Determine which image to use (dev or release) and check if already processed
	@if [ -n "$(VERSION)" ] && [ "$(STAGE)" = "release" ]; then \
		INIT_IMG="$(INIT_RELEASE_IMG):$(VERSION)"; \
	else \
		INIT_IMG="$(INIT_DRI_IMG):$(INIT_IMG_TAG)"; \
	fi; \
	if [ -f controllers/templates/lxd_initializer_ds.yaml.processed ]; then \
		if grep -q "$$INIT_IMG" controllers/templates/lxd_initializer_ds.yaml.processed; then \
			echo "Template already processed with image: $$INIT_IMG (skipping)"; \
			exit 0; \
		fi; \
	fi; \
	echo "Processing LXD initializer template with image: $$INIT_IMG"; \
	LXD_INITIALIZER_IMAGE=$$INIT_IMG envsubst '$$LXD_INITIALIZER_IMAGE' \
		< controllers/templates/lxd_initializer_ds.yaml > controllers/templates/lxd_initializer_ds.yaml.processed
	@# Verify the image was substituted
	@if [ -n "$(VERSION)" ] && [ "$(STAGE)" = "release" ]; then \
		VERIFY_IMG="$(INIT_RELEASE_IMG):$(VERSION)"; \
	else \
		VERIFY_IMG="$(INIT_DRI_IMG):$(INIT_IMG_TAG)"; \
	fi; \
	if ! grep -q "$$VERIFY_IMG" controllers/templates/lxd_initializer_ds.yaml.processed; then \
		if grep -q "\$${LXD_INITIALIZER_IMAGE}" controllers/templates/lxd_initializer_ds.yaml.processed; then \
			echo "ERROR: Image substitution failed! Still contains placeholder '\$${LXD_INITIALIZER_IMAGE}'"; \
			echo "Expected image: $$VERIFY_IMG"; \
			exit 1; \
		fi; \
	fi

.PHONY: generate-lxd-template
generate-lxd-template: controllers/templates/lxd_initializer_ds.yaml.processed ## Generate processed LXD initializer template for embedding
	@# Ensure processed file exists (required for go:embed)
	@if [ ! -f controllers/templates/lxd_initializer_ds.yaml.processed ]; then \
		echo "ERROR: Processed template not found"; \
		exit 1; \
	fi
	@echo "LXD initializer template ready for embedding"
