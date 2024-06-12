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
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var maasmachinetemplatelog = logf.Log.WithName("maasmachinetemplate-resource")

func (r *MaasMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachinetemplate,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachinetemplates,verbs=create;update,versions=v1beta1,name=mmaasmachinetemplate.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachinetemplate,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachinetemplates,versions=v1beta1,name=vmaasmachinetemplate.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1

var (
	_ webhook.Defaulter = &MaasMachineTemplate{}
	_ webhook.Validator = &MaasMachineTemplate{}
)

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MaasMachineTemplate) Default() {
	maasmachinetemplatelog.Info("default", "name", r.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateCreate() (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate update", "name", r.Name)
	oldM := old.(*MaasMachineTemplate)

	if r.Spec.Template.Spec.Image != oldM.Spec.Template.Spec.Image {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template image change is not allowed, old=%s, new=%s", oldM.Spec.Template.Spec.Image, r.Spec.Template.Spec.Image))
	}

	if *r.Spec.Template.Spec.MinCPU != *oldM.Spec.Template.Spec.MinCPU {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template min cpu count change is not allowed, old=%d, new=%d", oldM.Spec.Template.Spec.MinCPU, r.Spec.Template.Spec.MinCPU))
	}

	if *r.Spec.Template.Spec.MinMemoryInMB != *oldM.Spec.Template.Spec.MinMemoryInMB {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template min memory change is not allowed, old=%d MB, new=%d MB", oldM.Spec.Template.Spec.MinMemoryInMB, r.Spec.Template.Spec.MinMemoryInMB))
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateDelete() (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate delete", "name", r.Name)
	return nil, nil
}
