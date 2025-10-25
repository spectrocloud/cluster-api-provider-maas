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
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// HMCMaintenanceReconciler is a placeholder for the Host Maintenance Controller (HMC).
type HMCMaintenanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Namespace is the controller namespace (namespaced deployment)
	Namespace string
	// Recorder emits Kubernetes events
	Recorder record.EventRecorder
}

// Reconcile is a placeholder to enable wiring later.
func (r *HMCMaintenanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// First, try handling MaasMachine deletion flow (primary watch)
	mm := &infrav1beta1.MaasMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, mm); err == nil {
		if mm.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		// Add evacuation finalizer if missing
		if !controllerutil.ContainsFinalizer(mm, maint.HostEvacuationFinalizer) {
			controllerutil.AddFinalizer(mm, maint.HostEvacuationFinalizer)
			if err := r.Update(ctx, mm); err != nil {
				r.Log.Error(err, "add host-evacuation finalizer")
				return ctrl.Result{}, err
			}
			r.Log.Info("Added host evacuation finalizer", "maasMachine", req.NamespacedName.String())
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		// Determine host systemID
		if mm.Spec.SystemID == nil || *mm.Spec.SystemID == "" {
			// Without systemID we cannot proceed
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		hostSystemID := *mm.Spec.SystemID

		// Load/create session and tag host
		st, cm, err := maint.LoadSession(ctx, r.Client, r.Namespace)
		if err != nil {
			r.Log.Error(err, "load session")
			return ctrl.Result{}, err
		}
		if st.Status != maint.StatusActive || st.CurrentHost == "" || st.OpID == "" {
			st, err = maint.StartSession(ctx, r.Client, r.Namespace, hostSystemID)
			if err != nil {
				r.Log.Error(err, "start session")
				return ctrl.Result{}, err
			}
			r.Log.Info("HMC session started", "opId", st.OpID, "host", hostSystemID)
		}
		_ = cm

		// MAAS client from Secret
		maasClient, err := scope.NewMaasClientFromSecret(ctx, r.Client, r.Namespace, "maas-credentials")
		if err != nil {
			r.Log.Error(err, "failed to create MAAS client from Secret")
			return ctrl.Result{}, err
		}
		tags := maint.NewTagService(maasClient)
		inv := maint.NewInventoryService(maasClient)
		if err := maint.EnsureHostMaintenanceTags(tags, hostSystemID, st.OpID); err != nil {
			r.Log.Error(err, "ensure host maintenance tags", "host", hostSystemID)
			return ctrl.Result{}, err
		}
		r.Log.Info("host maintenance tags ensured", "host", hostSystemID, "opId", st.OpID)

		// Check evacuation gates
		svc := NewHostMaintenanceService(inv, tags, maasClient)
		ready, _, err := svc.CheckEvacuationGates(ctx, hostSystemID, st.OpID, r.Log)
		if err != nil {
			r.Log.Error(err, "check evacuation gates")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if !ready {
			r.Log.Info("Evacuation gates not yet satisfied; requeue", "host", hostSystemID)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		// Clear maintenance tags and remove finalizer to unblock deletion
		if err := svc.ClearMaintenanceTags(ctx, hostSystemID, st.OpID); err != nil {
			r.Log.Error(err, "clear maintenance tags", "host", hostSystemID)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		r.Recorder.Eventf(mm, corev1.EventTypeNormal, "HostMaintenanceCleared", "Cleared maintenance tags on host %s", hostSystemID)

		// TODO: Amit should be done in regulart maasmachine controller flow
		// // Best-effort deregister VM host if present
		// if vhosts, verr := maasClient.VMHosts().List(ctx, nil); verr == nil {
		// 	for _, vh := range vhosts {
		// 		if vh.HostSystemID() == hostSystemID {
		// 			_ = vh.Delete(ctx)
		// 			r.Log.Info("Deregistered VM host backing BM host", "host", hostSystemID)
		// 			break
		// 		}
		// 	}
		// }
		controllerutil.RemoveFinalizer(mm, maint.HostEvacuationFinalizer)
		if err := r.Update(ctx, mm); err != nil {
			r.Log.Error(err, "remove host-evacuation finalizer")
			return ctrl.Result{}, err
		}
		r.Log.Info("Host evacuation completed and finalizer removed", "host", hostSystemID)
		r.Recorder.Eventf(mm, corev1.EventTypeNormal, "EvacuationCompleted", "Evacuation completed; finalizer removed for host %s", hostSystemID)
		return ctrl.Result{}, nil
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	// Fallback: process ConfigMap triggered session tagging (best-effort)
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

	if st.Status != maint.StatusActive || st.CurrentHost == "" || st.OpID == "" {
		// No Active session: perform stale-tag GC
		maasClient, err := scope.NewMaasClientFromSecret(ctx, r.Client, r.Namespace, "maas-credentials")
		if err != nil {
			r.Log.Error(err, "failed to create MAAS client from Secret for GC")
			return ctrl.Result{}, nil
		}
		tags := maint.NewTagService(maasClient)
		inv := maint.NewInventoryService(maasClient)
		// Find hosts with maintenance tag
		machines, err := maasClient.Machines().List(ctx, maasclient.ParamsBuilder().Set(maasclient.TagKey, maint.TagHostMaintenance))
		if err == nil {
			for _, m := range machines {
				// Fetch details to access tags and systemID
				dm, gerr := m.Get(ctx)
				if gerr != nil {
					continue
				}
				sid := dm.SystemID()
				if sid == "" {
					continue
				}
				// Remove any stale op tags on the host
				if host, herr := inv.GetHost(sid); herr == nil {
					for _, t := range host.Tags {
						if strings.HasPrefix(t, maint.TagHostOpPrefix) {
							_ = tags.RemoveTagFromHost(sid, t)
						}
					}
				}
				// // Check host is empty via inventory of VM host
				// vms, lerr := inv.ListHostVMs(sid)
				// if lerr != nil {
				// 	continue
				// }
				// if len(vms) == 0 {
				// Clear maintenance and noschedule tags
				_ = tags.RemoveTagFromHost(sid, maint.TagHostNoSchedule)
				_ = tags.RemoveTagFromHost(sid, maint.TagHostMaintenance)
				r.Log.Info("stale maintenance tags cleared on empty host", "host", sid)
				// }
			}
		}
		return ctrl.Result{}, nil
	}

	// Build MAAS client from Secret in controller namespace
	maasClient, err := scope.NewMaasClientFromSecret(ctx, r.Client, r.Namespace, "maas-credentials")
	if err != nil {
		r.Log.Error(err, "failed to create MAAS client from Secret")
		return ctrl.Result{}, err
	}
	tags := maint.NewTagService(maasClient)

	// Ensure host tags
	if err := maint.EnsureHostMaintenanceTags(tags, st.CurrentHost, st.OpID); err != nil {
		r.Log.Error(err, "ensure host maintenance tags", "host", st.CurrentHost)
		return ctrl.Result{}, err
	}
	r.Log.Info("host maintenance tags ensured", "host", st.CurrentHost, "opId", st.OpID)
	return ctrl.Result{}, nil
}

// SetupWithManager is a placeholder registration; real watches will be added later.
func (r *HMCMaintenanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("hmc-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MaasMachine{}).
		For(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
