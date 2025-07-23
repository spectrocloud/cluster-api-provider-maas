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

package util

import (
	"context"
	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"regexp"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	// ErrEmptyProviderID means that the provider id is empty.
	ErrEmptyProviderID = errors.New("providerID is empty")

	// ErrInvalidProviderID means that the provider id has an invalid form.
	ErrInvalidProviderID = errors.New("providerID must be of the form <cloudProvider>://<optional>/<segments>/<provider id>")
)

// ProviderID is a struct representation of a Kubernetes ProviderID.
// Format: cloudProvider://optional/segments/etc/id
// For LXD VMs: maas-lxd:///zone/host_system_id/vm_name
type ProviderID struct {
	original      string
	cloudProvider string
	id            string
	isLXD         bool
	lxdHostID     string
	lxdVMName     string
	lxdZone       string
}

/*
- must start with at least one non-colon
- followed by ://
- followed by any number of characters
- must end with a non-slash, OR be a maas-lxd format.
*/
var providerIDRegex = regexp.MustCompile("^maas-lxd:///.*$|^[^:]+://.*[^/]$")

// GetMAASMachinesInCluster gets a cluster's MAASMachine resources.
func GetMAASMachinesInCluster(
	ctx context.Context,
	controllerClient client.Client,
	namespace, clusterName string) ([]*v1beta1.MaasMachine, error) {

	labels := map[string]string{clusterv1.ClusterNameLabel: clusterName}
	machineList := &v1beta1.MaasMachineList{}

	if err := controllerClient.List(
		ctx, machineList,
		client.InNamespace(namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	machines := make([]*v1beta1.MaasMachine, len(machineList.Items))
	for i := range machineList.Items {
		machines[i] = &machineList.Items[i]
	}

	return machines, nil
}

// NewProviderID parses the input string and returns a new ProviderID.
func NewProviderID(id string) (*ProviderID, error) {
	if id == "" {
		return nil, ErrEmptyProviderID
	}

	if !providerIDRegex.MatchString(id) {
		return nil, ErrInvalidProviderID
	}

	colonIndex := strings.Index(id, ":")
	cloudProvider := id[0:colonIndex]

	res := &ProviderID{
		original:      id,
		cloudProvider: cloudProvider,
	}

	// Handle LXD provider IDs differently
	if cloudProvider == "maas-lxd" {
		if err := res.parseLXDProviderID(id); err != nil {
			return nil, err
		}
	} else {
		// Traditional provider ID parsing
		lastSlashIndex := strings.LastIndex(id, "/")
		res.id = id[lastSlashIndex+1:]
	}

	if !res.Validate() {
		return nil, ErrInvalidProviderID
	}

	return res, nil
}

// parseLXDProviderID parses LXD-specific provider ID format
// Format: maas-lxd:///zone/host_system_id/vm_name
// Format with empty zone: maas-lxd:////host_system_id/vm_name
func (p *ProviderID) parseLXDProviderID(id string) error {
	p.isLXD = true

	// Remove the "maas-lxd:///" prefix
	path := strings.TrimPrefix(id, "maas-lxd:///")

	// Handle empty zone case: if path starts with "/", remove leading "/"
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}

	parts := strings.Split(path, "/")

	// Handle different cases based on number of parts
	if len(parts) == 2 {
		// Empty zone case: host_system_id/vm_name
		p.lxdZone = ""
		p.lxdHostID = parts[0]
		p.lxdVMName = parts[1]
	} else if len(parts) == 3 {
		// Normal case: zone/host_system_id/vm_name
		p.lxdZone = parts[0]
		p.lxdHostID = parts[1]
		p.lxdVMName = parts[2]
	} else {
		return ErrInvalidProviderID
	}

	p.id = p.lxdHostID // Use host system ID as the main ID for compatibility

	return nil
}

// CloudProvider returns the cloud provider portion of the ProviderID.
func (p *ProviderID) CloudProvider() string {
	return p.cloudProvider
}

// ID returns the identifier portion of the ProviderID.
// For LXD VMs, this returns the host system ID.
func (p *ProviderID) ID() string {
	return p.id
}

// IsLXD returns true if this is an LXD VM provider ID.
func (p *ProviderID) IsLXD() bool {
	return p.isLXD
}

// LXDHostID returns the host system ID for LXD VMs.
func (p *ProviderID) LXDHostID() string {
	return p.lxdHostID
}

// LXDVMName returns the VM name for LXD VMs.
func (p *ProviderID) LXDVMName() string {
	return p.lxdVMName
}

// LXDZone returns the availability zone for LXD VMs.
func (p *ProviderID) LXDZone() string {
	return p.lxdZone
}

// Equals returns true if this ProviderID string matches another ProviderID string.
func (p *ProviderID) Equals(o *ProviderID) bool {
	return p.String() == o.String()
}

// String returns the string representation of this object.
func (p ProviderID) String() string {
	return p.original
}

// Validate returns true if the provider id is valid.
func (p *ProviderID) Validate() bool {
	if p.CloudProvider() == "" || p.ID() == "" {
		return false
	}

	// Additional validation for LXD provider IDs
	if p.isLXD {
		return p.lxdHostID != "" && p.lxdVMName != ""
	}

	return true
}

// IndexKey returns the required level of uniqueness
// to represent and index machines uniquely from their node providerID.
func (p *ProviderID) IndexKey() string {
	return p.String()
}

// ParseLXDProviderID is a utility function to parse LXD provider IDs
// Returns zone, hostSystemID, vmName, and error
func ParseLXDProviderID(providerID string) (zone, hostSystemID, vmName string, err error) {
	if !strings.HasPrefix(providerID, "maas-lxd:///") {
		return "", "", "", errors.New("not an LXD provider ID")
	}

	pid, err := NewProviderID(providerID)
	if err != nil {
		return "", "", "", err
	}

	if !pid.IsLXD() {
		return "", "", "", errors.New("provider ID is not LXD format")
	}

	return pid.LXDZone(), pid.LXDHostID(), pid.LXDVMName(), nil
}

// IsLXDProviderID checks if a provider ID string is for an LXD VM
func IsLXDProviderID(providerID string) bool {
	return strings.HasPrefix(providerID, "maas-lxd:///")
}
