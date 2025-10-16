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

// SessionCMName is the name of the ConfigMap used to persist maintenance session state.
const SessionCMName = "hcp-maintenance-session"

// Status represents the lifecycle state of a maintenance session.
type Status string

const (
	StatusActive    Status = "Active"
	StatusCompleted Status = "Completed"
	StatusAborted   Status = "Aborted"
)

// State captures persisted fields for the current maintenance session.
type State struct {
	OpID        string
	Status      Status
	StartedAt   time.Time
	CurrentHost string
}

// MAAS API-facing interfaces and light types.
// TagService abstracts MAAS tag CRUD operations.
type TagService interface {
	// EnsureTag should be idempotent, create tags in MAAS if missing.
	EnsureTag(name string) error
	// AddTagToMachine adds a tag to a machine.
	AddTagToMachine(systemID, tag string) error
	// RemoveTagFromMachine removes a tag from a machine.
	RemoveTagFromMachine(systemID, tag string) error
	// AddTagToHost adds a tag to a host.
	AddTagToHost(systemID, tag string) error
	// RemoveTagFromHost removes a tag from a host.
	RemoveTagFromHost(systemID, tag string) error
}

// InventoryService abstracts MAAS inventory reads used by HMC/VEC.
type InventoryService interface {
	// ListHostVMs lists all VMs on a host.
	ListHostVMs(hostSystemID string) ([]Machine, error)
	// ResolveSystemIDByHostname resolves a hostname to a system ID.
	ResolveSystemIDByHostname(hostname string) (string, error)
	// GetMachine gets a machine by system ID.
	GetMachine(systemID string) (Machine, error)
	// GetHost returns the MAAS machine representing an LXD/KVM host (BM).
	GetHost(systemID string) (Machine, error)
	// GetVM returns the MAAS machine representing a VM.
	GetVM(systemID string) (Machine, error)
	// GetVMHostForVM resolves the host system ID for a given VM system ID.
	GetVMHostForVM(vmSystemID string) (hostSystemID string, err error)
}

// Machine is a minimal view of a MAAS machine/VM for maintenance flows.
type Machine struct {
	// SystemID is the unique identifier for the machine.
	SystemID string
	// HostSystemID is set for VMs and empty for hosts (BM/LXD hosts).
	HostSystemID string
	// Tags are the tags applied to the machine.
	Tags []string
	// Zone is the zone the machine is in.
	Zone string
	// FQDN is the fully qualified domain name of the machine.
	FQDN string
	// PowerState is the power state of the machine.
	PowerState string
	// PowerType is the power type of the machine.
	PowerType string
	// Hostname is the hostname of the machine.
	Hostname string
	// IPAddresses are the IP addresses of the machine.
	IPAddresses []string
}
