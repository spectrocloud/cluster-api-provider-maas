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

import "time"

// Session state stored in a ConfigMap; no CRD required.
const SessionCMName = "hcp-maintenance-session"

type Status string

const (
	StatusActive    Status = "Active"
	StatusCompleted Status = "Completed"
	StatusAborted   Status = "Aborted"
)

type State struct {
	OpID        string
	Status      Status
	StartedAt   time.Time
	CurrentHost string
}

// MAAS API-facing interfaces and light types.
type TagService interface {
	EnsureTag(name string) error
	AddTagToMachine(systemID, tag string) error
	RemoveTagFromMachine(systemID, tag string) error
	AddTagToHost(systemID, tag string) error
	RemoveTagFromHost(systemID, tag string) error
}

type InventoryService interface {
	ListHostVMs(hostSystemID string) ([]Machine, error)
	ResolveSystemIDByHostname(hostname string) (string, error)
	GetMachine(systemID string) (Machine, error)
}

type Machine struct {
	SystemID     string
	HostSystemID string
	Tags         []string
	Zone         string
}
