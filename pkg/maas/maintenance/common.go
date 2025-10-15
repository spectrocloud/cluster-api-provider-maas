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

import (
	"context"

	"github.com/spectrocloud/maas-client-go/maasclient"
)

// maasTagService implements the TagService interface using the MAAS client.
// Underlying it uses ClientSetInterface.
type maasTagService struct {
	client maasclient.ClientSetInterface
}

// NewTagService (constructor) creates a new TagService implementation.
// Passes the MAAS client into the service.
// This will be used to interact with methods instead of using direct struct instantiation.
func NewTagService(client maasclient.ClientSetInterface) TagService {
	return &maasTagService{
		client: client,
	}
}

// maasInventoryService implements the InventoryService interface using the MAAS client.
// Underlying it uses ClientSetInterface.
type maasInventoryService struct {
	client maasclient.ClientSetInterface
}

// NewInventoryService (constructor) creates a new InventoryService implementation.
// Passes the MAAS client into the service.
// This will be used to interact with methods instead of using direct struct instantiation.
func NewInventoryService(client maasclient.ClientSetInterface) InventoryService {
	return &maasInventoryService{
		client: client,
	}
}

func (s *maasTagService) EnsureTag(name string) error {
	// TODO: Implement
	return nil
}

// AddTagToHost adds a tag to a host machine identified by systemID.
func (s *maasTagService) AddTagToHost(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Assign(ctx, tag, []string{systemID})
}

// AddTagToMachine adds a tag to a machine identified by systemID.
func (s *maasTagService) AddTagToMachine(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Assign(ctx, tag, []string{systemID})
}

// RemoveTagFromMachine removes a tag from a machine identified by systemID.
func (s *maasTagService) RemoveTagFromMachine(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Unassign(ctx, tag, []string{systemID})
}

// RemoveTagFromHost removes a tag from a host machine identified by systemID.
func (s *maasTagService) RemoveTagFromHost(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Unassign(ctx, tag, []string{systemID})
}

// ListHostVMs lists all VMs running on the specified host.
func (s *maasInventoryService) ListHostVMs(hostSystemID string) ([]Machine, error) {
	ctx := context.Background()

	// Get the VMHost and list its machines
	vmHost := s.client.VMHosts().VMHost(hostSystemID)
	maasClientMachines, err := vmHost.Machines().List(ctx)
	if err != nil {
		return nil, err
	}

	// Convert MAAS Client Machines to maintenance.Machine objects
	var machinesForMaintenance []Machine
	for _, maasMachine := range maasClientMachines {
		// Get detailed machine info to access all fields
		detailedMachine, err := maasMachine.Get(ctx)
		if err != nil {
			// Skip machines we can't fetch details for
			continue
		}

		machinesForMaintenance = append(machinesForMaintenance, Machine{
			SystemID:     detailedMachine.SystemID(),
			HostSystemID: hostSystemID, // We know this from the input
			Tags:         detailedMachine.Tags(),
			Zone:         detailedMachine.ZoneName(),
		})
	}

	return machinesForMaintenance, nil
}

// ResolveSystemIDByHostname finds the system ID of a machine by its hostname.
func (s *maasInventoryService) ResolveSystemIDByHostname(hostname string) (string, error) {
	ctx := context.Background()

	// List all machines with hostname filter
	params := maasclient.ParamsBuilder().Set("hostname", hostname)
	machines, err := s.client.Machines().List(ctx, params)
	if err != nil {
		return "", err
	}

	// Find the first matching machine
	for _, machine := range machines {
		detailedMachine, err := machine.Get(ctx)
		if err != nil {
			continue
		}

		if detailedMachine.Hostname() == hostname {
			return detailedMachine.SystemID(), nil
		}
	}

	return "", nil // No machine found with this hostname
}

// GetMachine retrieves details about a specific machine by system ID.
func (s *maasInventoryService) GetMachine(systemID string) (Machine, error) {
	ctx := context.Background()

	// Get the machine from MAAS
	maasMachine := s.client.Machines().Machine(systemID)
	detailedMachine, err := maasMachine.Get(ctx)
	if err != nil {
		return Machine{}, err
	}

	// Note: HostSystemID is not directly available from the Machine object.
	// To get it, we'd need to list all VMHosts and find which one contains this machine.
	// For now, leaving it empty. Can be populated by the caller if needed.
	machine := Machine{
		SystemID:     detailedMachine.SystemID(),
		HostSystemID: "", // TODO: Need to query VMHosts to find parent host
		Tags:         detailedMachine.Tags(),
		Zone:         detailedMachine.ZoneName(),
	}

	return machine, nil
}
