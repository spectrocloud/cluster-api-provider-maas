ROOT_DIR_RELATIVE := ../..
include $(ROOT_DIR_RELATIVE)/common.mk

UNAME := $(shell uname -s)

# Directories.
BIN_DIR := bin
SHARE_DIR := share


## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(BIN_DIR):
	mkdir -p $@

$(SHARE_DIR):
	mkdir -p $@

MOCKGEN := $(BIN_DIR)/mockgen
$(MOCKGEN): $(BIN_DIR) go.mod go.sum # Build mockgen from tools folder.
	go build -tags=tools -o $@ github.com/golang/mock/mockgen


CONTROLLER_GEN := $(BIN_DIR)/controller-gen
$(CONTROLLER_GEN): $(BIN_DIR) go.mod go.sum # Build controller-gen from tools folder.
	go build -tags=tools -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

CONVERSION_GEN := $(BIN_DIR)/conversion-gen
$(CONVERSION_GEN): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/conversion-gen

DEFAULTER_GEN := $(BIN_DIR)/defaulter-gen
$(DEFAULTER_GEN): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/defaulter-gen

KUSTOMIZE := $(BIN_DIR)/kustomize ## Build kustomize
$(KUSTOMIZE): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ sigs.k8s.io/kustomize/kustomize/v3


.PHONY: clean
clean: ## Remove all tools
	rm -rf bin
	rm -rf share
