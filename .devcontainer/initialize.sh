#!/bin/bash

sudo chown -R vscode $HOME/.minikube
minikube start

echo DONE >> /tmp/foo