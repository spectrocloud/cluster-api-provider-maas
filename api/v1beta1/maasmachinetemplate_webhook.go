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
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var maasmachinetemplatelog = logf.Log.WithName("maasmachinetemplate-resource")

func (r *MaasMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachinetemplate,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachinetemplates,verbs=create;update,versions=v1beta1,name=mmaasmachinetemplate.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachinetemplate,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachinetemplates,versions=v1beta1,name=vmaasmachinetemplate.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1

var (
	_ admission.Defaulter[*MaasMachineTemplate] = &MaasMachineTemplate{}
	_ admission.Validator[*MaasMachineTemplate] = &MaasMachineTemplate{}
)

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *MaasMachineTemplate) Default(_ context.Context, _ *MaasMachineTemplate) error {
	maasmachinetemplatelog.Info("default", "name", r.Name)
	return nil
}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateCreate(_ context.Context, _ *MaasMachineTemplate) (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateUpdate(_ context.Context, oldObj, newObj *MaasMachineTemplate) (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate update", "name", newObj.Name)

	if newObj.Spec.Template.Spec.Image != oldObj.Spec.Template.Spec.Image {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template image change is not allowed, old=%s, new=%s", oldObj.Spec.Template.Spec.Image, newObj.Spec.Template.Spec.Image))
	}

	if *newObj.Spec.Template.Spec.MinCPU != *oldObj.Spec.Template.Spec.MinCPU {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template min cpu count change is not allowed, old=%d, new=%d", *oldObj.Spec.Template.Spec.MinCPU, *newObj.Spec.Template.Spec.MinCPU))
	}

	if *newObj.Spec.Template.Spec.MinMemoryInMB != *oldObj.Spec.Template.Spec.MinMemoryInMB {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine template min memory change is not allowed, old=%d MB, new=%d MB", *oldObj.Spec.Template.Spec.MinMemoryInMB, *newObj.Spec.Template.Spec.MinMemoryInMB))
	}
	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachineTemplate) ValidateDelete(_ context.Context, _ *MaasMachineTemplate) (admission.Warnings, error) {
	maasmachinetemplatelog.Info("validate delete", "name", r.Name)
	return nil, nil
}
