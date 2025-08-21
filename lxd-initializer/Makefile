# Image URL to use all building/pushing image targets
REGISTRY ?= us-east1-docker.pkg.dev/spectro-public/release/cluster-api-provider-maas
IMG ?= $(REGISTRY)/capmaas-lxd-initializer
TAG ?= latest

# Build binary
build:
	go build -o lxd-initializer lxd-initializer.go

# Run locally
run:
	./lxd-initializer

# Build the docker image
docker-build:
	docker build -t ${IMG}:${TAG} .

# Push the docker image
docker-push:
	docker push ${IMG}:${TAG}

# Build and push the docker image
docker-build-push: docker-build docker-push

# Deploy the DaemonSet
deploy:
	sed "s|\$${REGISTRY}|$(REGISTRY)|g; s|\$${TAG}|$(TAG)|g" lxd-initializer-daemonset.yaml | kubectl apply -f -

# Delete the DaemonSet
undeploy:
	kubectl delete -f lxd-initializer-daemonset.yaml --ignore-not-found=true

.PHONY: build run docker-build docker-push docker-build-push deploy undeploy 