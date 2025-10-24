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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
)

// HMCMaintenanceReconciler handles host maintenance operations via ConfigMap triggers
type HMCMaintenanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Namespace is the controller namespace (namespaced deployment)
	Namespace string
}

// Reconcile handles both MaasMachine evacuation finalizers and ConfigMap triggers
func (r *HMCMaintenanceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Namespace = request.Namespace

	// Try MaasMachine first
	var maasMachine infrav1beta1.MaasMachine
	if err := r.Get(ctx, request.NamespacedName, &maasMachine); err == nil {
		// Successfully got MaasMachine, handle evacuation
		return r.reconcileMaasMachine(ctx, &maasMachine)
	}

	// Try ConfigMap
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, request.NamespacedName, &configMap); err == nil {
		// Successfully got ConfigMap, handle maintenance triggers
		return r.reconcileConfigMap(ctx, request)
	}

	// Neither resource found, ignore
	r.Log.V(1).Info("No MaasMachine or ConfigMap found", "name", request.Name, "namespace", request.Namespace)
	return ctrl.Result{}, nil
}

// reconcileConfigMap handles ConfigMap reconciliation (existing logic)
func (r *HMCMaintenanceReconciler) reconcileConfigMap(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
	maasClient := maint.NewMAASClient(r.Client, r.Namespace)
	tags := maint.NewTagService(maasClient)
	if err := maint.EnsureHostMaintenanceTags(tags, st.CurrentHost, st.OpID); err != nil {
		r.Log.Error(err, "ensure host maintenance tags", "host", st.CurrentHost)
		return ctrl.Result{}, err
	}
	r.Log.Info("host maintenance tags ensured", "host", st.CurrentHost, "opId", st.OpID)
	return ctrl.Result{}, nil
}

// reconcileMaasMachine handles MaasMachine reconciliation for evacuation finalizers
func (r *HMCMaintenanceReconciler) reconcileMaasMachine(ctx context.Context, maasMachine *infrav1beta1.MaasMachine) (ctrl.Result, error) {
	log := r.Log.WithValues("maasmachine", maasMachine.Name)

	// Only process host machines (not VMs) with evacuation finalizer
	if maasMachine.Spec.Parent != nil && *maasMachine.Spec.Parent != "" {
		return ctrl.Result{}, nil // This is a VM, skip
	}

	if !controllerutil.ContainsFinalizer(maasMachine, HostEvacuationFinalizer) {
		return ctrl.Result{}, nil // No evacuation finalizer, skip
	}

	// Only process deletion events
	if maasMachine.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil // Not being deleted, skip
	}

	log.Info("Processing host evacuation for MaasMachine deletion")

	// Create host maintenance service
	hmcService := NewHostMaintenanceService(r.Client, maasMachine.Namespace)

	// Check evacuation gates
	evacuationReady, err := hmcService.CheckEvacuationGates(ctx, maasMachine, log)
	if err != nil {
		log.Error(err, "failed to check evacuation gates")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	if !evacuationReady {
		log.Info("Evacuation gates not met, blocking deletion",
			"host", maasMachine.Spec.SystemID,
			"requeueAfter", 10*time.Second)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Evacuation gates are met, clear tags and remove finalizer
	if err := hmcService.ClearMaintenanceTagsAndRemoveFinalizer(ctx, maasMachine, log); err != nil {
		log.Error(err, "failed to clear maintenance tags and remove finalizer")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	log.Info("Host evacuation completed successfully")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *HMCMaintenanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MaasMachine{}).
		For(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
