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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows MaasMachineReconciler to clean up resources associated with MaasMachine before
	// removing it from the apiserver.
	MachineFinalizer = "maasmachine.infrastructure.cluster.x-k8s.io"
)

// ProvisioningMode defines how the machine should be provisioned
// +kubebuilder:validation:Enum=baremetal;lxd
type ProvisioningMode string

const (
	// ProvisioningModeBaremetal provisions directly on bare metal
	ProvisioningModeBaremetal ProvisioningMode = "baremetal"
	// ProvisioningModeLXD provisions as an LXD VM inside a bare metal host
	ProvisioningModeLXD ProvisioningMode = "lxd"
)

// LXDConfig defines configuration for LXD VM provisioning
type LXDConfig struct {
	// HostSelection defines how to select LXD hosts
	// +optional
	HostSelection *LXDHostSelectionConfig `json:"hostSelection,omitempty"`

	// ResourceAllocation defines resources for the LXD VM
	// +optional
	ResourceAllocation *LXDResourceConfig `json:"resourceAllocation,omitempty"`

	// StorageConfig defines storage configuration for the LXD VM
	// +optional
	StorageConfig *LXDStorageConfig `json:"storageConfig,omitempty"`

	// NetworkConfig defines network configuration for the LXD VM
	// +optional
	NetworkConfig *LXDNetworkConfig `json:"networkConfig,omitempty"`
}

// LXDHostSelectionConfig defines how to select LXD hosts
type LXDHostSelectionConfig struct {
	// PreferredHosts lists preferred LXD hosts by system ID
	// +optional
	PreferredHosts []string `json:"preferredHosts,omitempty"`

	// AvailabilityZones lists preferred availability zones for host selection
	// +optional
	AvailabilityZones []string `json:"availabilityZones,omitempty"`

	// ResourcePools lists preferred resource pools for host selection
	// +optional
	ResourcePools []string `json:"resourcePools,omitempty"`

	// Tags for host selection
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// LXDResourceConfig defines resource allocation for LXD VMs
type LXDResourceConfig struct {
	// CPU allocation in cores
	// +kubebuilder:validation:Minimum=1
	// +optional
	CPU *int `json:"cpu,omitempty"`

	// Memory allocation in MB
	// +kubebuilder:validation:Minimum=512
	// +optional
	Memory *int `json:"memory,omitempty"`

	// Disk size in GB
	// +kubebuilder:validation:Minimum=1
	// +optional
	Disk *int `json:"disk,omitempty"`
}

// LXDStorageConfig defines storage configuration for LXD VMs
type LXDStorageConfig struct {
	// StoragePool specifies which LXD storage pool to use
	// +optional
	StoragePool *string `json:"storagePool,omitempty"`
}

// LXDNetworkConfig defines network configuration for LXD VMs
type LXDNetworkConfig struct {
	// Bridge specifies which bridge to connect the VM to
	// +optional
	Bridge *string `json:"bridge,omitempty"`

	// MacAddress allows specifying a custom MAC address
	// +optional
	MacAddress *string `json:"macAddress,omitempty"`

	// StaticIPConfig defines static IP configuration for the LXD VM
	// +optional
	StaticIPConfig *LXDStaticIPConfig `json:"staticIPConfig,omitempty"`
}

// LXDStaticIPConfig defines static IP assignment configuration for LXD VMs
type LXDStaticIPConfig struct {
	// IPAddress is the static IP address to assign to the VM
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`
	IPAddress string `json:"ipAddress"`

	// Gateway is the default gateway for the static IP configuration
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`
	Gateway string `json:"gateway"`

	// Subnet defines the subnet mask or CIDR notation for the static IP
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)|\/([0-9]|[1-2][0-9]|3[0-2])$`
	Subnet string `json:"subnet"`

	// DNSServers is a list of DNS server IP addresses
	// +optional
	DNSServers []string `json:"dnsServers,omitempty"`

	// Interface specifies the network interface to configure with static IP
	// If not specified, the default interface will be used
	// +optional
	Interface *string `json:"interface,omitempty"`
}

// MaasMachineSpec defines the desired state of MaasMachine
type MaasMachineSpec struct {

	// FailureDomain is the failure domain the machine will be created in.
	// Must match a key in the FailureDomains map stored on the cluster object.
	// +optional
	FailureDomain *string `json:"failureDomain,omitempty"`

	// SystemID will be the MaaS machine ID
	// +optional
	SystemID *string `json:"systemID,omitempty"`

	// ProviderID will be the name in ProviderID format (maas://<zone>/system_id)
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// ResourcePool will be the MAAS Machine resourcepool
	// +optional
	ResourcePool *string `json:"resourcePool,omitempty"`

	// MinCPU minimum number of CPUs
	// +kubebuilder:validation:Minimum=0
	MinCPU *int `json:"minCPU"`

	// MinMemoryInMB minimum memory in MB
	// +kubebuilder:validation:Minimum=0
	MinMemoryInMB *int `json:"minMemory"`

	// Tags for placement
	// +optional
	Tags []string `json:"tags,omitempty"`

	// Image will be the MaaS image id
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// ProvisioningMode defines how this machine should be provisioned
	// Defaults to "baremetal" for backward compatibility
	// +kubebuilder:default=baremetal
	// +optional
	ProvisioningMode ProvisioningMode `json:"provisioningMode,omitempty"`

	// LXDConfig provides configuration for LXD VM provisioning
	// Only used when ProvisioningMode is "lxd"
	// +optional
	LXDConfig *LXDConfig `json:"lxdConfig,omitempty"`
}

// LXDMachineStatus defines the observed state of an LXD VM
type LXDMachineStatus struct {
	// HostSystemID is the system ID of the host machine running the LXD VM
	HostSystemID *string `json:"hostSystemID,omitempty"`

	// VMName is the name of the LXD VM
	VMName *string `json:"vmName,omitempty"`

	// HostAddress is the IP address of the LXD host
	HostAddress *string `json:"hostAddress,omitempty"`

	// ResourceAllocation shows the actual resources allocated to the LXD VM
	ResourceAllocation *LXDResourceConfig `json:"resourceAllocation,omitempty"`

	// NetworkStatus contains the actual network configuration of the LXD VM
	// +optional
	NetworkStatus *LXDNetworkStatus `json:"networkStatus,omitempty"`
}

// LXDNetworkStatus represents the actual network configuration status of an LXD VM
type LXDNetworkStatus struct {
	// AssignedIP is the actual IP address assigned to the VM
	// +optional
	AssignedIP *string `json:"assignedIP,omitempty"`

	// Interface is the network interface name that has the assigned IP
	// +optional
	Interface *string `json:"interface,omitempty"`

	// ConfigMethod indicates how the IP was configured ("dhcp" or "static")
	ConfigMethod string `json:"configMethod"`

	// Gateway is the gateway IP address being used
	// +optional
	Gateway *string `json:"gateway,omitempty"`

	// DNSServers is the list of DNS servers configured for the VM
	// +optional
	DNSServers []string `json:"dnsServers,omitempty"`
}

// MaasMachineStatus defines the observed state of MaasMachine
type MaasMachineStatus struct {

	// Ready denotes that the machine (maas container) is ready
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// MachineState is the state of this MAAS machine.
	MachineState *MachineState `json:"machineState,omitempty"`

	// MachinePowered is if the machine is "Powered" on
	MachinePowered bool `json:"machinePowered,omitempty"`

	// Hostname is the actual MaaS hostname
	Hostname *string `json:"hostname,omitempty"`

	// DNSAttached specifies whether the DNS record contains the IP of this machine
	DNSAttached bool `json:"dnsAttached,omitempty"`

	// Addresses contains the associated addresses for the maas machine.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// Conditions defines current service state of the MaasMachine.
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	FailureMessage *string `json:"failureMessage,omitempty"`

	// LXDMachine contains status information for LXD VM provisioning
	// Only populated when ProvisioningMode is "lxd"
	// +optional
	LXDMachine *LXDMachineStatus `json:"lxdMachine,omitempty"`
}

// +kubebuilder:resource:path=maasmachines,scope=Namespaced,categories=cluster-api
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// MaasMachine is the Schema for the maasmachines API
type MaasMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaasMachineSpec   `json:"spec,omitempty"`
	Status MaasMachineStatus `json:"status,omitempty"`
}

func (c *MaasMachine) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *MaasMachine) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// MaasMachineList contains a list of MaasMachine
type MaasMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaasMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaasMachine{}, &MaasMachineList{})
}
