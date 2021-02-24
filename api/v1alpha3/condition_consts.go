/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha3

import clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"

// Conditions and condition Reasons for the DockerMachine object

const (
	// MachineDeployedCondition documents the status of the deployment of a machine

	MachineDeployedCondition clusterv1.ConditionType = "MachineDeployed"

	// WaitingForClusterInfrastructureReason (Severity=Info) documents a MachineMachine waiting for the cluster
	// infrastructure to be ready before starting to deploy the machine that provides the MachineMachine
	// infrastructure.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	// WaitingForBootstrapDataReason (Severity=Info) documents a MachineMachine waiting for the bootstrap
	// script to be ready before starting to create the container that provides the MachineMachine infrastructure.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"

	// MachineDeployingReason
	MachineDeployingReason = "MachineDeploying"

	// MachineTerminatedReason
	MachineTerminatedReason = "MachineTerminatedReason"

	// MachineDeployingReason
	MachinePoweredOffReason = "MachinePoweredOff"

	// MachineNotFoundReason used when the machine couldn't be retrieved.
	MachineNotFoundReason = "MachineNotFound"

	// MachineDeployFailedReason (Severity=Warning) documents a MachineMachine controller detecting
	// an error while deploying the MaaS machine that provides the MachineMachine infrastructure; those kind of
	// errors are usually transient and failed provisioning are automatically re-tried by the controller.
	MachineDeployFailedReason = "MachineDeployFailed"

	// MachineDeployStartedReason (Severity=Info) documents a MachineMachine controller started deploying
	MachineDeployStartedReason = "MachineDeployStartedReason"
)

//const (
//	// BootstrapExecSucceededCondition provides an observation of the DockerMachine bootstrap process.
//	// 	It is set based on successful execution of bootstrap commands and on the existence of
//	//	the /run/cluster-api/bootstrap-success.complete file.
//	// The condition gets generated after ContainerProvisionedCondition is True.
//	//
//	// NOTE as a difference from other providers, container provisioning and bootstrap are directly managed
//	// by the DockerMachine controller (not by cloud-init).
//	BootstrapExecSucceededCondition clusterv1.ConditionType = "BootstrapExecSucceeded"
//
//	// BootstrappingReason documents (Severity=Info) a DockerMachine currently executing the bootstrap
//	// script that creates the Kubernetes node on the newly provisioned machine infrastructure.
//	BootstrappingReason = "Bootstrapping"
//
//	// BootstrapFailedReason documents (Severity=Warning) a DockerMachine controller detecting an error while
//	// bootstrapping the Kubernetes node on the machine just provisioned; those kind of errors are usually
//	// transient and failed bootstrap are automatically re-tried by the controller.
//	BootstrapFailedReason = "BootstrapFailed"
//)
//
// Conditions and condition Reasons for the DockerCluster object

const (
	// Only applicable to control plane machines. DNSAttachedCondition will report true when a control plane is successfully registered with an DNS
	// When set to false, severity can be an Error if the subnet is not found or unavailable in the instance's AZ
	DNSAttachedCondition clusterv1.ConditionType = "DNSAttached"

	DNSDetachPending = "DNSDetachPending"
	DNSAttachPending = "DNSAttachPending"
)

// Cluster Conditions

const (
	// DNSReadyCondition documents the availability of the container that implements the cluster DNS.
	//
	DNSReadyCondition clusterv1.ConditionType = "LoadBalancerReady"

	// LoadBalancerProvisioningFailedReason (Severity=Warning) documents a DockerCluster controller detecting
	// an error while provisioning the container that provides the cluster load balancer.; those kind of
	// errors are usually transient and failed provisioning are automatically re-tried by the controller.
	DNSFailedReason = "LoadBalancerFailed"

	WaitForDNSNameReason = "WaitForDNSName"
)

const (
	// APIServerAvailableCondition documents whether API server is reachable
	APIServerAvailableCondition clusterv1.ConditionType = "APIServerAvailable"

	// APIServerNotReadyReason api server isn't responding
	APIServerNotReadyReason = "APIServerNotReady"
)
