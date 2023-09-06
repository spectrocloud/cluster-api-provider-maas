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
