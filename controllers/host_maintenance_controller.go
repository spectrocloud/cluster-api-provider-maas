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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
)

const (
	// HostEvacuationFinalizer is the finalizer that blocks deletion until evacuation criteria are met
	HostEvacuationFinalizer = "maas.lxd.io/host-evacuation"

	// ReconcileInterval is the interval for reconciling HMC operations
	ReconcileInterval = 30 * time.Second

	// EvacuationCheckInterval is the interval for checking evacuation gates
	// Prioritizing WLC lifecycle over HCP cluster upgrade flow
	EvacuationCheckInterval = 10 * time.Second
)

// HMCMaintenanceReconciler handles host maintenance operations including evacuation finalizers
type HMCMaintenanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Namespace is the controller namespace (namespaced deployment)
	Namespace string

	// Services for MAAS operations
	tagService       maint.TagService
	inventoryService maint.InventoryService
}

// Reconcile handles both ConfigMap triggers and MaasMachine finalizer operations
func (r *HMCMaintenanceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Namespace = request.Namespace

	// Initialize services if not already done
	if r.tagService == nil {
		maasClient := maint.NewMAASClient(r.Client, r.Namespace)
		r.tagService = maint.NewTagService(maasClient)
		r.inventoryService = maint.NewInventoryService(maasClient)
	}

	// Check if this is a MaasMachine reconciliation
	var maasMachine infrav1beta1.MaasMachine
	if err := r.Get(ctx, request.NamespacedName, &maasMachine); err == nil {
		return r.reconcileMaasMachine(ctx, &maasMachine)
	}

	// Otherwise, handle ConfigMap reconciliation (existing logic)
	return r.reconcileConfigMap(ctx, request)
}

// reconcileMaasMachine handles MaasMachine reconciliation for evacuation finalizers
func (r *HMCMaintenanceReconciler) reconcileMaasMachine(ctx context.Context, maasMachine *infrav1beta1.MaasMachine) (ctrl.Result, error) {
	log := r.Log.WithValues("maasmachine", maasMachine.Name)

	// Only process host machines (not VMs)
	if !r.isHostMachine(maasMachine) {
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if !maasMachine.DeletionTimestamp.IsZero() {
		return r.reconcileMaasMachineDelete(ctx, maasMachine, log)
	}

	// Handle normal reconciliation
	return r.reconcileMaasMachineNormal(ctx, maasMachine, log)
}

// reconcileMaasMachineNormal handles normal reconciliation of MaasMachine
func (r *HMCMaintenanceReconciler) reconcileMaasMachineNormal(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) (ctrl.Result, error) {
	// Check if finalizer needs to be added
	if !containsString(maasMachine.Finalizers, HostEvacuationFinalizer) {
		log.Info("Adding host evacuation finalizer")
		maasMachine.Finalizers = append(maasMachine.Finalizers, HostEvacuationFinalizer)
		if err := r.Update(ctx, maasMachine); err != nil {
			log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Perform stale tag garbage collection
	if err := r.performStaleTagGC(ctx, maasMachine, log); err != nil {
		log.Error(err, "failed to perform stale tag garbage collection")
		// Don't return error, just log it
	}

	return ctrl.Result{RequeueAfter: ReconcileInterval}, nil
}

// reconcileMaasMachineDelete handles deletion of MaasMachine
func (r *HMCMaintenanceReconciler) reconcileMaasMachineDelete(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) (ctrl.Result, error) {
	log.Info("Processing host deletion with evacuation finalizer")

	// Check evacuation gates
	evacuationReady, err := r.checkEvacuationGates(ctx, maasMachine, log)
	if err != nil {
		log.Error(err, "failed to check evacuation gates")
		return ctrl.Result{RequeueAfter: EvacuationCheckInterval}, err
	}

	if !evacuationReady {
		log.Info("Evacuation gates not met, blocking deletion",
			"host", maasMachine.Spec.SystemID,
			"requeueAfter", EvacuationCheckInterval)
		return ctrl.Result{RequeueAfter: EvacuationCheckInterval}, nil
	}

	// Evacuation gates are met, proceed with cleanup
	log.Info("Evacuation gates met, proceeding with host cleanup")

	// Clear maintenance tags
	if err := r.clearMaintenanceTags(ctx, maasMachine, log); err != nil {
		log.Error(err, "failed to clear maintenance tags")
		return ctrl.Result{RequeueAfter: EvacuationCheckInterval}, err
	}

	// Deregister host (if needed)
	if err := r.deregisterHost(ctx, maasMachine, log); err != nil {
		log.Error(err, "failed to deregister host")
		return ctrl.Result{RequeueAfter: EvacuationCheckInterval}, err
	}

	// Remove finalizer
	log.Info("Removing host evacuation finalizer")
	maasMachine.Finalizers = removeString(maasMachine.Finalizers, HostEvacuationFinalizer)
	if err := r.Update(ctx, maasMachine); err != nil {
		log.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Host evacuation completed successfully")
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

// SetupWithManager sets up the controller with the Manager
func (r *HMCMaintenanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MaasMachine{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return r.isHostMachine(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return r.isHostMachine(e.ObjectNew)
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return r.isHostMachine(e.Object)
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return r.isHostMachine(e.Object)
			},
		}).
		For(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

// isHostMachine checks if the MaasMachine is a host (not a VM)
func (r *HMCMaintenanceReconciler) isHostMachine(obj client.Object) bool {
	maasMachine, ok := obj.(*infrav1beta1.MaasMachine)
	if !ok {
		return false
	}

	// Check if this is a host machine (not a VM)
	// VMs typically have a parent reference, hosts don't
	return maasMachine.Spec.Parent == nil || *maasMachine.Spec.Parent == ""
}

// checkEvacuationGates checks if evacuation criteria are met
func (r *HMCMaintenanceReconciler) checkEvacuationGates(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) (bool, error) {
	if maasMachine.Spec.SystemID == nil {
		return false, fmt.Errorf("maasMachine has no systemID")
	}
	hostSystemID := *maasMachine.Spec.SystemID

	// Gate 1: Check if host is empty (no VMs running)
	hostEmpty, err := r.isHostEmpty(ctx, hostSystemID, log)
	if err != nil {
		return false, fmt.Errorf("failed to check if host is empty: %w", err)
	}

	if !hostEmpty {
		log.Info("Host not empty, evacuation blocked", "host", hostSystemID)
		return false, nil
	}

	// Gate 2: Check per-WLC ready-op-<uuid> tags on CP VMs
	wlcReady, err := r.checkWLCReadyTags(ctx, hostSystemID, log)
	if err != nil {
		return false, fmt.Errorf("failed to check WLC ready tags: %w", err)
	}

	if !wlcReady {
		log.Info("WLC ready tags not met, evacuation blocked", "host", hostSystemID)
		return false, nil
	}

	log.Info("All evacuation gates met", "host", hostSystemID)
	return true, nil
}

// isHostEmpty checks if the host has no running VMs
func (r *HMCMaintenanceReconciler) isHostEmpty(ctx context.Context, hostSystemID string, log logr.Logger) (bool, error) {
	vms, err := r.inventoryService.ListHostVMs(hostSystemID)
	if err != nil {
		return false, fmt.Errorf("failed to list host VMs: %w", err)
	}

	// Check if any VMs are in running state
	for _, vm := range vms {
		if vm.PowerState == "on" || vm.PowerState == "running" {
			log.Info("Host has running VM", "host", hostSystemID, "vm", vm.SystemID, "powerState", vm.PowerState)
			return false, nil
		}
	}

	log.Info("Host is empty", "host", hostSystemID, "vmCount", len(vms))
	return true, nil
}

// checkWLCReadyTags checks if all WLC clusters have ready-op-<uuid> tags
func (r *HMCMaintenanceReconciler) checkWLCReadyTags(ctx context.Context, hostSystemID string, log logr.Logger) (bool, error) {
	// Get all VMs on the host
	vms, err := r.inventoryService.ListHostVMs(hostSystemID)
	if err != nil {
		return false, fmt.Errorf("failed to list host VMs: %w", err)
	}

	// Check each VM for ready-op-<uuid> tags
	for _, vm := range vms {
		vmDetails, err := r.inventoryService.GetVM(vm.SystemID)
		if err != nil {
			log.Error(err, "failed to get VM details", "vm", vm.SystemID)
			continue
		}

		// Check if VM has any ready-op-<uuid> tags
		hasReadyTag := false
		for _, tag := range vmDetails.Tags {
			if len(tag) > len(maint.TagHostOpPrefix) && tag[:len(maint.TagHostOpPrefix)] == maint.TagHostOpPrefix {
				hasReadyTag = true
				break
			}
		}

		if !hasReadyTag {
			log.Info("VM missing ready-op tag", "vm", vm.SystemID, "tags", vmDetails.Tags)
			return false, nil
		}
	}

	log.Info("All VMs have ready-op tags", "host", hostSystemID)
	return true, nil
}

// clearMaintenanceTags clears maintenance-related tags from the host
func (r *HMCMaintenanceReconciler) clearMaintenanceTags(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) error {
	if maasMachine.Spec.SystemID == nil {
		return fmt.Errorf("maasMachine has no systemID")
	}
	hostSystemID := *maasMachine.Spec.SystemID

	// Clear maintenance tags
	maintenanceTags := []string{
		maint.TagHostMaintenance,
		maint.TagHostNoSchedule,
	}

	for _, tag := range maintenanceTags {
		if err := r.tagService.RemoveTagFromHost(hostSystemID, tag); err != nil {
			log.Error(err, "failed to remove maintenance tag", "tag", tag, "host", hostSystemID)
			return err
		}
		log.Info("Cleared maintenance tag", "tag", tag, "host", hostSystemID)
	}

	return nil
}

// deregisterHost deregisters the host from MAAS (if needed)
func (r *HMCMaintenanceReconciler) deregisterHost(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) error {
	// TODO: Implement host deregistration logic
	// This might involve calling MAAS API to release the machine
	log.Info("Host deregistration not yet implemented", "host", maasMachine.Spec.SystemID)
	return nil
}

// performStaleTagGC performs garbage collection of stale tags
func (r *HMCMaintenanceReconciler) performStaleTagGC(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) error {
	if maasMachine.Spec.SystemID == nil {
		return fmt.Errorf("maasMachine has no systemID")
	}
	hostSystemID := *maasMachine.Spec.SystemID

	// Check if host is empty and has no active maintenance session
	hostEmpty, err := r.isHostEmpty(ctx, hostSystemID, log)
	if err != nil {
		return err
	}

	if !hostEmpty {
		return nil // Host not empty, skip GC
	}

	// Check if host has maintenance tags but no active session
	hostDetails, err := r.inventoryService.GetHost(hostSystemID)
	if err != nil {
		return err
	}

	hasMaintenanceTags := false
	for _, tag := range hostDetails.Tags {
		if tag == maint.TagHostMaintenance || tag == maint.TagHostNoSchedule {
			hasMaintenanceTags = true
			break
		}
	}

	if hasMaintenanceTags {
		log.Info("Clearing stale maintenance tags from empty host", "host", hostSystemID)
		return r.clearMaintenanceTags(ctx, maasMachine, log)
	}

	return nil
}

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
