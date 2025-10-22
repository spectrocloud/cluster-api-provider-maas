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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
)

// HMCMaintenanceReconciler is a placeholder for the Host Maintenance Controller (HMC).
type HMCMaintenanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Namespace is the controller namespace (namespaced deployment)
	Namespace string
}

// Reconcile is a placeholder to enable wiring later.
func (r *HMCMaintenanceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Namespace = request.Namespace
	// Load or create session
	st, cm, err := maint.LoadSession(ctx, r.Client, r.Namespace)
	if err != nil {
		r.Log.Error(err, "load session")
		return ctrl.Result{}, err
	}
	// Optional external trigger via CM
	if start, host := maint.ShouldStartFromTrigger(cm); start {
		st, err = maint.StartSession(ctx, r.Client, r.Namespace, host)
		if err != nil {
			r.Log.Error(err, "start session")
			return ctrl.Result{}, err
		}
		r.Log.Info("HMC session started from trigger", "opId", st.OpID, "host", host)
	}
	// No active session: nothing to do
	if st.Status != maint.StatusActive || st.CurrentHost == "" || st.OpID == "" {
		return ctrl.Result{}, nil
	}

	// Tag host with maintenance markers using real MAAS client
	maasClient := scope.NewMaasClient(nil)
	tags := maint.NewTagService(maasClient)
	if err := maint.EnsureHostMaintenanceTags(tags, st.CurrentHost, st.OpID); err != nil {
		r.Log.Error(err, "ensure host maintenance tags", "host", st.CurrentHost)
		return ctrl.Result{}, err
	}
	r.Log.Info("host maintenance tags ensured", "host", st.CurrentHost, "opId", st.OpID)
	return ctrl.Result{}, nil
}

// SetupWithManager is a placeholder registration; real watches will be added later.
func (r *HMCMaintenanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Minimal registration for scaffolding; real watches to be added in PCP-5339.
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
