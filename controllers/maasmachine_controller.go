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
	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/machine"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/controllers/remote"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
)

// MaasMachineReconciler reconciles a MaasMachine object
type MaasMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Tracker  *remote.ClusterCacheTracker
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasmachines/status,verbs=get;update;patch

func (r *MaasMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("maasmachine", req.NamespacedName)

	// Fetch the MaasMachine instance.
	maasMachine := &infrav1.MaasMachine{}
	err := r.Get(ctx, req.NamespacedName, maasMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, maasMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, maasMachine) {
		log.Info("MaasMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// Get Infra cluster
	maasCluster := &infrav1.MaasCluster{}
	infraClusterName := client.ObjectKey{
		Namespace: maasMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, infraClusterName, maasCluster); err != nil {
		log.Info("MaasCluster is not available yet")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:         r.Client,
		Logger:         log,
		Cluster:        cluster,
		MaasCluster:    maasCluster,
		ControllerName: "maasmachine",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:       log,
		Client:       r.Client,
		Tracker:      r.Tracker,
		Cluster:      cluster,
		ClusterScope: clusterScope,
		Machine:      machine,
		MaasMachine:  maasMachine,
	})
	if err != nil {
		log.Error(err, "failed to create scope")
		return ctrl.Result{}, err
	}

	// Always close the scope when exiting this function so we can persist any MaasMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Handle deleted machines
	if !maasCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope, clusterScope)
}

func (r *MaasMachineReconciler) reconcileDelete(_ context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	machineScope.Info("Reconciling MaasMachine delete")

	maasMachine := machineScope.MaasMachine

	// TODO(saamalik)
	fmt.Println(clusterScope)

	// Machine is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(maasMachine, infrav1.MachineFinalizer)

	return reconcile.Result{}, nil
}

// findInstance queries the EC2 apis and retrieves the instance if it exists, returns nil otherwise.
func (r *MaasMachineReconciler) findMachine(machineScope *scope.MachineScope, machineSvc *machine.Service) (*infrav1.Machine, error) {
	id := machineScope.GetInstanceID()
	if id == nil {
		return nil, nil
	}

	m, err := machineSvc.GetMachine(*id)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to find machine")
	}

	return m, nil
}

func (r *MaasMachineReconciler) reconcileNormal(_ context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	machineScope.Info("Reconciling MaasMachine")

	maasMachine := machineScope.MaasMachine

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(maasMachine, infrav1.MachineFinalizer) {
		controllerutil.AddFinalizer(maasMachine, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.WaitingForBootstrapDataReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	machineSvc := machine.NewService(machineScope)

	// Find existing instance
	machine, err := r.findMachine(machineScope, machineSvc)
	if err != nil {
		machineScope.Error(err, "unable to find machine")
		conditions.MarkUnknown(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.MachineNotFoundReason, err.Error())
		return ctrl.Result{}, err
	}

	// Create new machine
	// TODO(saamalik) confirm that we'll never "recreate" a machine; e.g: findMachine should always return err
	// if there used to be a machine
	if machine == nil {
		// Avoid a flickering condition between Started and Failed if there's a persistent failure with createInstance
		if conditions.GetReason(machineScope.MaasMachine, infrav1.MachineDeployedCondition) != infrav1.MachineDeployFailedReason {
			conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.MachineDeployStartedReason, clusterv1.ConditionSeverityInfo, "")
			if patchErr := machineScope.PatchObject(); patchErr != nil {
				machineScope.Error(patchErr, "failed to patch conditions")
				return ctrl.Result{}, patchErr
			}
		}
		machine, err = r.deployMachine( /*ec2svc, */ machineScope, clusterScope)
		if err != nil {
			machineScope.Error(err, "unable to create machine")
			conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.MachineDeployFailedReason, clusterv1.ConditionSeverityError, err.Error())
			return ctrl.Result{}, err
		}
	}

	// Make sure Spec.ProviderID and Spec.InstanceID are always set.
	machineScope.SetProviderID(machine.ID, machine.AvailabilityZone)
	machineScope.SetSystemID(machine.ID)
	machineScope.SetMachineHostname(machine.Hostname)

	existingMachineState := machineScope.GetMachineState()
	machineScope.SetMachineState(machine.State)

	// Proceed to reconcile the AWSMachine state.
	if existingMachineState == nil || *existingMachineState != machine.State {
		machineScope.Info("MaaS machine state changed", "state", machine.State, "system-id", *machineScope.GetInstanceID())
	}

	switch s := machine.State; {
	case !machine.Powered:
		machineScope.SetNotReady()
		conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.MachinePoweredOffReason, clusterv1.ConditionSeverityWarning, "")
	case s == infrav1.MachineStateDeploying, s == infrav1.MachineStateAllocated:
		machineScope.SetNotReady()
		conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.MachineDeployingReason, clusterv1.ConditionSeverityWarning, "")
	//case infrav1.InstanceStateStopping, infrav1.InstanceStateStopped:
	//	machineScope.SetNotReady()
	//	conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.InstanceStoppedReason, clusterv1.ConditionSeverityError, "")
	case s == infrav1.MachineStateDeployed:
		machineScope.SetReady()
		conditions.MarkTrue(machineScope.MaasMachine, infrav1.MachineDeployedCondition)
	//case infrav1.InstanceStateShuttingDown, infrav1.InstanceStateTerminated:
	//	machineScope.SetNotReady()
	//	machineScope.Info("Unexpected EC2 machine termination", "state", machine.State, "machine-id", *machineScope.GetInstanceID())
	//	r.Recorder.Eventf(machineScope.MaasMachine, corev1.EventTypeWarning, "InstanceUnexpectedTermination", "Unexpected EC2 machine termination")
	//	conditions.MarkFalse(machineScope.MaasMachine, infrav1.MachineDeployedCondition, infrav1.InstanceTerminatedReason, clusterv1.ConditionSeverityError, "")
	default:
		machineScope.SetNotReady()
		machineScope.Info("MaaS machine state is undefined", "state", machine.State, "system-id", *machineScope.GetInstanceID())
		r.Recorder.Eventf(machineScope.MaasMachine, corev1.EventTypeWarning, "InstanceUnhandledState", "MaaS machine state is undefined")
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("MaaS machine state %q is undefined", machine.State))
		conditions.MarkUnknown(machineScope.MaasMachine, infrav1.MachineDeployedCondition, "", "")
	}

	// TODO(saamalik) when terminated
	//if instance.State == infrav1.InstanceStateTerminated {
	//	machineScope.SetFailureReason(capierrors.UpdateMachineError)
	//	machineScope.SetFailureMessage(errors.Errorf("EC2 instance state %q is unexpected", instance.State))
	//}

	// tasks that can take place during all known instance states
	if machineScope.MachineIsInKnownState() {
		// TODO(saamalik) tags / labels
		//_, err = r.ensureTags(ec2svc, machineScope.AWSMachine, machineScope.GetInstanceID(), machineScope.AdditionalTags())
		//if err != nil {
		//	machineScope.Error(err, "failed to ensure tags")
		//	return ctrl.Result{}, err
		//}

		if err := r.reconcileLBAttachment(machineScope, clusterScope, machine); err != nil {
			machineScope.Error(err, "failed to reconcile LB attachment")
			return ctrl.Result{}, err
		}
	}

	// tasks that can only take place during operational instance states
	if machineScope.MachineIsOperational() {
		machineScope.SetAddresses(machine.Addresses)
		if err := machineScope.SetNodeProviderID(); err != nil {
			machineScope.Error(err, "Unable to set Node hostname")
			r.Recorder.Eventf(machineScope.MaasMachine, corev1.EventTypeWarning, "NodeProviderUpdateFailed", "Unable to set the node provider update")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MaasMachineReconciler) deployMachine(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*infrav1.Machine, error) {
	machineScope.Info("Deploying on MaaS machine")

	userData, userDataErr := r.resolveUserData(machineScope)
	if userDataErr != nil {
		return nil, errors.Wrapf(userDataErr, "failed to resolve userdata")
	}

	// TODO REMOVE
	fmt.Println(userData)
	fmt.Println(clusterScope)

	//instance, err := ec2svc.CreateInstance(machineScope, userData)
	//if err != nil {
	//	return nil, errors.Wrapf(err, "failed to create MaasMachine instance")
	//}

	return nil, nil
}

func (r *MaasMachineReconciler) resolveUserData(machineScope *scope.MachineScope) (string, error) {
	userData, err := machineScope.GetRawBootstrapData()
	if err != nil {
		r.Recorder.Eventf(machineScope.MaasMachine, corev1.EventTypeWarning, "FailedGetBootstrapData", err.Error())
		return "", err
	}

	return string(userData), nil
}

func (r *MaasMachineReconciler) reconcileLBAttachment(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope, _ *infrav1.Machine) error {
	if !machineScope.IsControlPlane() {
		return nil
	}

	// https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/95104b9b793d394a50dc2e95d98200c8398713ed/controllers/awsmachine_controller.go#L716

	// MachinePowered
	// MachineState

	// TODO
	fmt.Println(clusterScope)

	conditions.MarkTrue(machineScope.MaasMachine, infrav1.DNSAttachedCondition)
	return nil
}

// SetupWithManager will add watches for this controller
func (r *MaasMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	clusterToMaasMachines, err := util.ClusterToObjectsMapper(mgr.GetClient(), &infrav1.MaasMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.MaasMachine{}).
		WithOptions(options).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("MaasMachine"))),
		).
		Watches(
			&source.Kind{Type: &infrav1.MaasCluster{}},
			handler.EnqueueRequestsFromMapFunc(r.MaasClusterToMaasMachines),
		).
		Build(r)
	if err != nil {
		return err
	}
	return c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToMaasMachines),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	)
}

// MaasClusterToMaasMachines is a handler.ToRequestsFunc to be used to enqeue
// requests for reconciliation of MaasMachines.
func (r *MaasMachineReconciler) MaasClusterToMaasMachines(o client.Object) []ctrl.Request {
	var result []ctrl.Request
	c, ok := o.(*infrav1.MaasCluster)
	if !ok {
		panic(fmt.Sprintf("Expected a MaasCluster but got a %T", o))
	}

	cluster, err := util.GetOwnerCluster(context.TODO(), r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		return result
	}

	labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(context.TODO(), machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil
	}
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}
