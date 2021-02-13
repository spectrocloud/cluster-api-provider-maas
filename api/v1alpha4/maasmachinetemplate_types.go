/*
Copyright 2019 The Kubernetes Authors.

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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MaasMachineTemplateSpec defines the desired state of MaasMachineTemplate
type MaasMachineTemplateSpec struct {
	Template MaasMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=maasmachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// MaasMachineTemplate is the Schema for the maasmachinetemplates API
type MaasMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MaasMachineTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// MaasMachineTemplateList contains a list of MaasMachineTemplate
type MaasMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaasMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaasMachineTemplate{}, &MaasMachineTemplateList{})
}

// MaasMachineTemplateResource describes the data needed to create a MaasMachine from a template
type MaasMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec MaasMachineSpec `json:"spec"`
}
