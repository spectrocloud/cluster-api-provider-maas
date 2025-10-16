/*
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
package maintenance

// TriggerSource identifies what started a maintenance session.
type TriggerSource int

const (
	// TriggerKCP is the trigger source for a maintenance session started by the KCP.
	TriggerKCP TriggerSource = iota
	// TriggerHostDelete is the trigger source for a maintenance session started by the host delete.
	TriggerHostDelete
)

// SessionManager manages OpID lifecycle for a maintenance session.
// Implementations should persist the active session to allow resume after restart.
type SessionManager interface {
	// StartIfNone starts a new maintenance session if none is active.
	StartIfNone(trigger TriggerSource) (opID string, started bool, err error)
	// Current returns the current maintenance session.
	Current() (opID string, active bool, err error)
	// Complete completes the current maintenance session.
	Complete() error
}

// HostEvacuation defines host-level operations for evacuation and gating.
// TagHost must write host maintenance/NoSchedule and OpID tags.
// GatesSatisfied checks both: host empty and per-WLC ready-op-<opID> acks.
type HostEvacuation interface {
	// TagHost writes host maintenance/NoSchedule and OpID tags.
	TagHost(hostSystemID, opID string) error
	// ComputeImpactedClusters computes the clusters that are impacted by the evacuation.
	ComputeImpactedClusters(hostSystemID string) ([]string, error)
	// GatesSatisfied checks if the evacuation gates are satisfied.
	GatesSatisfied(hostSystemID, opID string, impactedClusters []string) (bool, error)
	// ClearAndDeregister clears the host evacuation tags and deregisters the host.
	ClearAndDeregister(hostSystemID string) error
}

// HostEvacuationFinalizer prevents host deletion until evacuation gates pass.
// HostEvacuationFinalizer prevents host deletion until evacuation gates pass.
// Use dashes to comply with MAAS and upstream naming conventions.
const HostEvacuationFinalizer = "maas-lxd-host-evacuation"

// ControlPlaneSelector selects the CP Machine scheduled on the maintenance host.
type ControlPlaneSelector interface {
	// FindCPMachineOnHost finds the CP machine on the host.
	FindCPMachineOnHost(hostSystemID string) (machineID string, err error)
}

// CPReplacer replaces the targeted CP Machine either by delete-first (3-CP)
// or by template swap with maxSurge=1 (1-CP).
type CPReplacer interface {
	// DeleteCPMachine deletes the CP machine.
	DeleteCPMachine(machineID string) error
	// TemplateSwapOne swaps the CP machine with a template.
	TemplateSwapOne() error
}

// ReadinessChecker verifies KCP, Node, and API readiness before ready tagging.
type ReadinessChecker interface {
	// IsKCPConverged checks if the KCP is converged.
	IsKCPConverged() (bool, error)
	// IsNodeReady checks if the node is ready.
	IsNodeReady(nodeName string) (bool, error)
	// IsAPIReady checks if the API is ready.
	IsAPIReady() (bool, error)
}

// ReadyTagger writes ready-op-<opID> on the replacement CP VM and may also
// ensure cp and wlc-<clusterId> tags are present.
type ReadyTagger interface {
	// TagReplacement writes ready-op-<opID> on the replacement CP VM and may also
	// ensure cp and wlc-<clusterId> tags are present.
	TagReplacement(systemID, opID, clusterID string) error
}
