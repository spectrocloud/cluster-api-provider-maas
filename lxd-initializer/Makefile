# Variables
REGISTRY ?= spectrocloud
TAG ?= latest
IMAGE_NAME = lxd-initializer

# Go build flags
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS = -v

# Docker build flags
DOCKER_BUILD_FLAGS = --platform linux/$(GOARCH)

.PHONY: all build docker-build docker-push clean

all: build docker-build

build:
	@echo "Building LXD initializer binary..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GO_BUILD_FLAGS) -o lxd-initializer lxd-initializer.go

docker-build: build
	@echo "Building Docker image..."
	docker build $(DOCKER_BUILD_FLAGS) -t $(REGISTRY)/$(IMAGE_NAME):$(TAG) .

docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push $(REGISTRY)/$(IMAGE_NAME):$(TAG)

clean:
	@echo "Cleaning up..."
	rm -f lxd-initializer 