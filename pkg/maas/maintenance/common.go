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
	"fmt"
	"net"
	"os"

	"github.com/spectrocloud/maas-client-go/maasclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// EnsureTagInInventory ensures a tag exists in MAAS inventory (idempotent).
// If the tag already exists, this is a no-op. If it doesn't exist, it creates it.
func (s *maasTagService) EnsureTagInInventory(name string) error {
	ctx := context.Background()

	// List all existing tags
	existingTags, err := s.client.Tags().List(ctx)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, tag := range existingTags {
		if tag.Name() == name {
			// Tag already exists, return success (idempotent)
			return nil
		}
	}

	// Tag doesn't exist, create it
	return s.client.Tags().Create(ctx, name)
}

// AddTagToHost adds a tag to a host machine identified by systemID.
func (s *maasTagService) AddTagToHost(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Assign(ctx, tag, systemID)
}

// AddTagToMachine adds a tag to a machine identified by systemID.
func (s *maasTagService) AddTagToMachine(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Assign(ctx, tag, systemID)
}

// RemoveTagFromMachine removes a tag from a machine identified by systemID.
func (s *maasTagService) RemoveTagFromMachine(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Unassign(ctx, tag, systemID)
}

// RemoveTagFromHost removes a tag from a host machine identified by systemID.
func (s *maasTagService) RemoveTagFromHost(systemID, tag string) error {
	ctx := context.Background()
	return s.client.Tags().Unassign(ctx, tag, systemID)
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
			ResourcePool: detailedMachine.ResourcePoolName(),
			FQDN:         detailedMachine.FQDN(),
			PowerState:   detailedMachine.PowerState(),
			PowerType:    detailedMachine.PowerType(),
			Hostname:     detailedMachine.Hostname(),
			IPAddresses:  convertIPAddresses(detailedMachine.IPAddresses()),
		})
	}

	return machinesForMaintenance, nil
}

// ListAllVMs lists all VMs in the MAAS inventory (machines with power_type="lxd")
func (s *maasInventoryService) ListAllVMs() ([]Machine, error) {
	ctx := context.Background()

	// List all machines in MAAS
	allMachines, err := s.client.Machines().List(ctx, maasclient.ParamsBuilder())
	if err != nil {
		return nil, err
	}

	// Filter for VMs (power_type="lxd") and convert to maintenance.Machine objects
	var vms []Machine
	for _, maasMachine := range allMachines {
		// Get detailed machine info to access all fields including power type
		detailedMachine, err := maasMachine.Get(ctx)
		if err != nil {
			// Skip machines we can't fetch details for
			continue
		}

		// Only include LXD VMs
		if detailedMachine.PowerType() != "lxd" {
			continue
		}

		vms = append(vms, Machine{
			SystemID:     detailedMachine.SystemID(),
			HostSystemID: detailedMachine.Parent(), // Parent system_id for LXD VMs
			Tags:         detailedMachine.Tags(),
			Zone:         detailedMachine.ZoneName(),
			ResourcePool: detailedMachine.ResourcePoolName(),
			FQDN:         detailedMachine.FQDN(),
			PowerState:   detailedMachine.PowerState(),
			PowerType:    detailedMachine.PowerType(),
			Hostname:     detailedMachine.Hostname(),
			IPAddresses:  convertIPAddresses(detailedMachine.IPAddresses()),
		})
	}

	return vms, nil
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
// This method automatically detects if the machine is a VM and populates the HostSystemID accordingly.
func (s *maasInventoryService) GetMachine(systemID string) (Machine, error) {
	ctx := context.Background()

	// Get the machine from MAAS
	maasMachine := s.client.Machines().Machine(systemID)
	detailedMachine, err := maasMachine.Get(ctx)
	if err != nil {
		return Machine{}, err
	}

	// Use Parent() method to get parent system_id for LXD VMs
	// Returns empty string for non-VM machines (bare metal hosts)
	machine := Machine{
		SystemID:     detailedMachine.SystemID(),
		HostSystemID: detailedMachine.Parent(), // Automatically populated for LXD VMs
		Tags:         detailedMachine.Tags(),
		Zone:         detailedMachine.ZoneName(),
		ResourcePool: detailedMachine.ResourcePoolName(),
		FQDN:         detailedMachine.FQDN(),
		PowerState:   detailedMachine.PowerState(),
		PowerType:    detailedMachine.PowerType(),
		Hostname:     detailedMachine.Hostname(),
		IPAddresses:  convertIPAddresses(detailedMachine.IPAddresses()),
	}

	return machine, nil
}

// GetHost retrieves details about a host (bare metal) machine by system ID.
func (s *maasInventoryService) GetHost(systemID string) (Machine, error) {
	ctx := context.Background()

	// Get the machine from MAAS
	maasMachine := s.client.Machines().Machine(systemID)
	detailedMachine, err := maasMachine.Get(ctx)
	if err != nil {
		return Machine{}, err
	}

	// A host is a bare metal machine, so HostSystemID should be empty
	machine := Machine{
		SystemID:     detailedMachine.SystemID(),
		HostSystemID: "", // Hosts don't have a parent host
		Tags:         detailedMachine.Tags(),
		Zone:         detailedMachine.ZoneName(),
		ResourcePool: detailedMachine.ResourcePoolName(),
		FQDN:         detailedMachine.FQDN(),
		PowerState:   detailedMachine.PowerState(),
		PowerType:    detailedMachine.PowerType(),
		Hostname:     detailedMachine.Hostname(),
		IPAddresses:  convertIPAddresses(detailedMachine.IPAddresses()),
	}

	return machine, nil
}

// GetVM retrieves details about a VM by system ID.
// This is semantically the same as GetMachine but explicitly indicates this is a VM.
func (s *maasInventoryService) GetVM(systemID string) (Machine, error) {
	ctx := context.Background()

	// Get the machine from MAAS
	maasMachine := s.client.Machines().Machine(systemID)
	detailedMachine, err := maasMachine.Get(ctx)
	if err != nil {
		return Machine{}, err
	}

	// Use Parent() method to get parent system_id for LXD VMs
	machine := Machine{
		SystemID:     detailedMachine.SystemID(),
		HostSystemID: detailedMachine.Parent(), // Parent system_id for LXD VMs
		Tags:         detailedMachine.Tags(),
		Zone:         detailedMachine.ZoneName(),
		ResourcePool: detailedMachine.ResourcePoolName(),
		FQDN:         detailedMachine.FQDN(),
		PowerState:   detailedMachine.PowerState(),
		PowerType:    detailedMachine.PowerType(),
		Hostname:     detailedMachine.Hostname(),
		IPAddresses:  convertIPAddresses(detailedMachine.IPAddresses()),
	}

	return machine, nil
}

// GetVMHostForVM resolves the host system ID for a given VM system ID.
// This method uses the Parent() method which returns the parent system_id for LXD VMs.
func (s *maasInventoryService) GetVMHostForVM(vmSystemID string) (string, error) {
	ctx := context.Background()

	// Get the VM machine details
	maasMachine := s.client.Machines().Machine(vmSystemID)
	detailedMachine, err := maasMachine.Get(ctx)
	if err != nil {
		return "", err
	}

	// Use the Parent() method which returns parent system_id for LXD VMs
	// Returns empty string if not an LXD VM
	parentSystemID := detailedMachine.Parent()

	return parentSystemID, nil
}

// convertIPAddresses converts []net.IP to []string
func convertIPAddresses(ips []net.IP) []string {
	var result []string
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result
}

// NewMAASClient creates a new MAAS client using the capmaas-manager-bootstrap-credentials secret
// Falls back to environment variables if the secret is not available
// Returns error if credentials cannot be found
func NewMAASClient(k8sClient client.Client, namespace string) (maasclient.ClientSetInterface, error) {
	// Get MAAS credentials using the same logic as ClusterScope.GetMaasClientIdentity
	endpoint, apiKey, err := GetMAASCredentials(k8sClient, namespace)
	if err != nil {
		return nil, err
	}

	client := maasclient.NewAuthenticatedClientSet(endpoint, apiKey)
	return client, nil
}

// GetMAASCredentials retrieves MAAS credentials from the capmaas-manager-bootstrap-credentials secret
// Falls back to environment variables if the secret is not available
// This function implements the same logic as ClusterScope.GetMaasClientIdentity but as a standalone function
func GetMAASCredentials(k8sClient client.Client, namespace string) (endpoint, apiKey string, err error) {
	// Secret containing MAAS endpoint/token created by Palette bootstrapper
	secretName := "capmaas-manager-bootstrap-credentials"

	// Try to get credentials from secret first
	if k8sClient != nil && namespace != "" {
		secret := &corev1.Secret{}
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      secretName,
		}

		// Try to get the secret
		if err := k8sClient.Get(context.Background(), key, secret); err == nil {
			// Get the credentials from the secret
			endpoint = string(secret.Data["MAAS_ENDPOINT"])
			apiKey = string(secret.Data["MAAS_API_KEY"])

			// If both credentials are valid, return them
			if endpoint != "" && apiKey != "" {
				return endpoint, apiKey, nil
			}
		}
	}

	// Fall back to environment variables
	endpoint = os.Getenv("MAAS_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("MAAS_API_URL") // Alternative env var name
	}

	apiKey = os.Getenv("MAAS_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("MAAS_API_TOKEN") // Alternative env var name
	}

	// Validate that we have both required credentials
	if endpoint == "" {
		return "", "", fmt.Errorf("MAAS endpoint not found: check capmaas-manager-bootstrap-credentials secret (MAAS_ENDPOINT) or MAAS_ENDPOINT/MAAS_API_URL environment variable")
	}

	if apiKey == "" {
		return "", "", fmt.Errorf("MAAS API key not found: check capmaas-manager-bootstrap-credentials secret (MAAS_API_KEY) or MAAS_API_KEY/MAAS_API_TOKEN environment variable")
	}

	return endpoint, apiKey, nil
}
