package v1alpha4

import (
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

// MachineState describes the state of an AWS instance.
type MachineState string

var (
	//MachineStateDeploying is the string representing an instance in a deploying state
	MachineStateDeploying = MachineState("Deploying")

	// MachineStateRunning is the string representing an instance in a pending state
	MachineStateDeployed = MachineState("Deployed")

	// MachineStateReady is the string representing an instance in a ready (commissioned) state
	MachineStateReady = MachineState("Ready")

	// MachineStateAllocated is the string representing an instance in a ready (commissioned) state
	MachineStateAllocated = MachineState("Allocated")

	//// MachineStateShuttingDown is the string representing an instance shutting down
	//MachineStateShuttingDown = MachineState("shutting-down")
	//
	//// MachineStateTerminated is the string representing an instance that has been terminated
	//MachineStateTerminated = MachineState("terminated")
	//
	//// MachineStateStopping is the string representing an instance
	//// that is in the process of being stopped and can be restarted
	//MachineStateStopping = MachineState("stopping")

	// MachineStateStopped is the string representing an instance
	// that has been stopped and can be restarted
	//MachineStateStopped = MachineState("stopped")

	// MachineRunningStates defines the set of states in which an MaaS instance is
	// running or going to be running soon
	MachineRunningStates = sets.NewString(
		string(MachineStateDeploying),
		string(MachineStateDeployed),
	)

	// MachineOperationalStates defines the set of states in which an MaaS instance is
	// or can return to running, and supports all MaaS operations
	MachineOperationalStates = MachineRunningStates.Union(
		sets.NewString(
			string(MachineStateAllocated),
			//string(MachineStateStopped),
		),
	)

	// MachineKnownStates represents all known MaaS instance states
	MachineKnownStates = MachineOperationalStates.Union(
		sets.NewString(
			string(MachineStateReady),
			//string(MachineStateTerminated),
		),
	)
)

// Instance describes an AWS instance.
type Machine struct {
	ID string

	// The current state of the machine.
	State MachineState

	// The current state of the machine.
	Powered bool

	// The AZ of the machine
	AvailabilityZone string

	// Addresses contains the AWS instance associated addresses.
	Addresses []clusterv1.MachineAddress
}
