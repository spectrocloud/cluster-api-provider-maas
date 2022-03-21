#!/bin/bash

# sudo chown -R vscode $HOME/.minikube
# minikube start
# echo "start minikube"
echo "start kind cluster"
pwd

# delete older (if running)
kind delete cluster --name kind-maas

# create kind
kind create cluster --config .devcontainer/kind-cluster-with-extramounts.yaml --wait 300s

# install docker
clusterctl init --infrastructure docker

# generate fun
clusterctl generate cluster docker1 --infrastructure=docker --control-plane-machine-count=1 --worker-machine-count=1 --kubernetes-version=1.21.10 --flavor development > docker-simple.yaml
