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
	TriggerKCP TriggerSource = iota
	TriggerHostDelete
)

// SessionManager manages opId lifecycle for a maintenance session.
type SessionManager interface {
	StartIfNone(trigger TriggerSource) (opID string, started bool, err error)
	Current() (opID string, active bool, err error)
	Complete() error
}

// HostEvacuation defines host-level operations for evacuation and gating.
type HostEvacuation interface {
	TagHost(hostSystemID, opID string) error
	ComputeImpactedClusters(hostSystemID string) ([]string, error)
	GatesSatisfied(hostSystemID, opID string, impactedClusters []string) (bool, error)
	ClearAndDeregister(hostSystemID string) error
}

const HostEvacuationFinalizer = "maas.lxd-host-evacuation"

// Control-plane related contracts for VEC.
type ControlPlaneSelector interface {
	FindCPMachineOnHost(hostSystemID string) (machineID string, err error)
}

type CPReplacer interface {
	DeleteCPMachine(machineID string) error
	TemplateSwapOne() error
}

type ReadinessChecker interface {
	IsKCPConverged() (bool, error)
	IsNodeReady(nodeName string) (bool, error)
	IsAPIReady() (bool, error)
}

type ReadyTagger interface {
	TagReplacement(systemID, opID, clusterID string) error
}
