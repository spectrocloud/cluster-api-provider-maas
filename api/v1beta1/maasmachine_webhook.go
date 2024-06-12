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
var maasmachinelog = logf.Log.WithName("maasmachine-resource")

func (r *MaasMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,verbs=create;update,versions=v1beta1,name=mmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,versions=v1beta1,name=vmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1

var (
	_ webhook.Defaulter = &MaasMachine{}
	_ webhook.Validator = &MaasMachine{}
)

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MaasMachine) Default() {
	maasmachinelog.Info("default", "name", r.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateCreate() (admission.Warnings, error) {
	maasmachinelog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateDelete() (admission.Warnings, error) {
	maasmachinelog.Info("validate delete", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	maasmachinelog.Info("validate update", "name", r.Name)
	oldM := old.(*MaasMachine)

	if r.Spec.Image != oldM.Spec.Image {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine image change is not allowed, old=%s, new=%s", oldM.Spec.Image, r.Spec.Image))
	}

	if *r.Spec.MinCPU != *oldM.Spec.MinCPU {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min cpu count change is not allowed, old=%d, new=%d", oldM.Spec.MinCPU, r.Spec.MinCPU))
	}

	if *r.Spec.MinMemoryInMB != *oldM.Spec.MinMemoryInMB {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min memory change is not allowed, old=%d MB, new=%d MB", oldM.Spec.MinMemoryInMB, r.Spec.MinMemoryInMB))
	}
	return nil, nil
}
