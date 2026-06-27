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
var maasmachinelog = logf.Log.WithName("maasmachine-resource")

func (r *MaasMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,verbs=create;update,versions=v1beta1,name=mmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,versions=v1beta1,name=vmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1

var (
	_ admission.Defaulter[*MaasMachine] = &MaasMachine{}
	_ admission.Validator[*MaasMachine] = &MaasMachine{}
)

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *MaasMachine) Default(_ context.Context, _ *MaasMachine) error {
	maasmachinelog.Info("default", "name", r.Name)
	return nil
}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateCreate(_ context.Context, _ *MaasMachine) (admission.Warnings, error) {
	maasmachinelog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateDelete(_ context.Context, _ *MaasMachine) (admission.Warnings, error) {
	maasmachinelog.Info("validate delete", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateUpdate(_ context.Context, oldObj, newObj *MaasMachine) (admission.Warnings, error) {
	maasmachinelog.Info("validate update", "name", newObj.Name)

	if newObj.Spec.Image != oldObj.Spec.Image {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine image change is not allowed, old=%s, new=%s", oldObj.Spec.Image, newObj.Spec.Image))
	}

	if *newObj.Spec.MinCPU != *oldObj.Spec.MinCPU {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min cpu count change is not allowed, old=%d, new=%d", *oldObj.Spec.MinCPU, *newObj.Spec.MinCPU))
	}

	if *newObj.Spec.MinMemoryInMB != *oldObj.Spec.MinMemoryInMB {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min memory change is not allowed, old=%d MB, new=%d MB", *oldObj.Spec.MinMemoryInMB, *newObj.Spec.MinMemoryInMB))
	}
	return nil, nil
}
