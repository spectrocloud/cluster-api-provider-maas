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
	"net"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/dns"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	infrautil "github.com/spectrocloud/cluster-api-provider-maas/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// MaasClusterReconciler reconciles a MaasCluster object
type MaasClusterReconciler struct {
	client.Client
	Log                 logr.Logger
	Scheme              *runtime.Scheme
	Recorder            record.EventRecorder
	GenericEventChannel chan event.GenericEvent
	Tracker             *remote.ClusterCacheTracker
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasclusters/status,verbs=get;update;patch

// Reconcile reads that state of the cluster for a MaasCluster object and makes changes based on the state read
// and what is in the MaasCluster.Spec
func (r *MaasClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("maascluster", req.Name)

	// Fetch the MaasCluster instance
	maasCluster := &infrav1beta1.MaasCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, maasCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, maasCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on MaasCluster")
		return ctrl.Result{}, nil
	}

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:              r.Client,
		Logger:              log,
		Cluster:             cluster,
		MaasCluster:         maasCluster,
		ClusterEventChannel: r.GenericEventChannel,
		ControllerName:      "maascluster",
		Tracker:             r.Tracker,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any MAAS Cluster changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Support FailureDomains
	// In cloud providers this would likely look up which failure domains are supported and set the status appropriately.
	// so kCP will distribute the CPs across multiple failure domains
	failureDomains := make(clusterv1.FailureDomains)
	for _, az := range maasCluster.Spec.FailureDomains {
		failureDomains[az] = clusterv1.FailureDomainSpec{
			ControlPlane: true,
		}
	}
	maasCluster.Status.FailureDomains = failureDomains

	// Handle deleted clusters
	if !maasCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, clusterScope)
}

func (r *MaasClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	clusterScope.Info("Reconciling MaasCluster delete")

	maasCluster := clusterScope.MaasCluster

	maasMachines, err := infrautil.GetMAASMachinesInCluster(ctx, r.Client, clusterScope.Cluster.Namespace, clusterScope.Cluster.Name)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err,
			"unable to list MAASMachines part of MAASCluster %s/%s", clusterScope.Cluster.Namespace, clusterScope.Cluster.Name)
	}

	if len(maasMachines) > 0 {
		r.Log.Info("Waiting for MAASMachines to be deleted", "count", len(maasMachines))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(maasCluster, infrav1beta1.ClusterFinalizer)

	// TODO(saamalik) implement the recorder stuff (look at aws)

	return reconcile.Result{}, nil
}

func (r *MaasClusterReconciler) reconcileDNSAttachments(clusterScope *scope.ClusterScope, dnssvc *dns.Service) error {

	if clusterScope.IsCustomEndpoint() {
		return nil
	}

	machines, err := clusterScope.GetClusterMaasMachines()
	if err != nil {
		return errors.Wrapf(err, "Unable to list all maas machines")
	}

	preferredSubnets, err := clusterScope.GetPreferredSubnets()
	if err != nil {
		return errors.Wrapf(err, "unable to find preferred subnets")
	}

	var runningIpAddresses []string

	currentIPs, err := dnssvc.GetAPIServerDNSRecords()
	if err != nil {
		return errors.Wrap(err, "Unable to get the dns resources")
	}

	machinesPendingAttachment := make([]*infrav1beta1.MaasMachine, 0)
	machinesPendingDetachment := make([]*infrav1beta1.MaasMachine, 0)

	for _, m := range machines {
		if !IsControlPlaneMachine(m) {
			continue
		}

		machineIP := getExternalMachineIP(clusterScope.Logger, preferredSubnets, m)
		attached := currentIPs.Has(machineIP)
		isRunningHealthy := IsRunning(m)

		if !m.DeletionTimestamp.IsZero() || !isRunningHealthy {
			if attached {
				clusterScope.Info("Cleaning up IP on unhealthy machine", "machine", m.Name)
				machinesPendingDetachment = append(machinesPendingDetachment, m)
			}
		} else if IsRunning(m) {
			if !attached {
				clusterScope.Info("Healthy machine without DNS attachment; attaching.", "machine", m.Name)
				machinesPendingAttachment = append(machinesPendingAttachment, m)
			}

			runningIpAddresses = append(runningIpAddresses, machineIP)
		}
		//r.Recorder.Eventf(machineScope.MaasMachine, corev1.EventTypeNormal, "SuccessfulDetachControlPlaneDNS",
		//	"Control plane instance %q is de-registered from load balancer", i.ID)
		//runningIpAddresses = append(runningIpAddresses, m.)
	}

	if err := dnssvc.UpdateDNSAttachments(runningIpAddresses); err != nil {
		return err
	} else if len(machinesPendingAttachment) > 0 || len(machinesPendingDetachment) > 0 {
		clusterScope.Info("Pending DNS attachments or detachments; will retry again")
		return ErrRequeueDNS
	}

	return nil
}

// IsControlPlaneMachine checks machine is a control plane node.
func IsControlPlaneMachine(m *infrav1beta1.MaasMachine) bool {
	_, ok := m.ObjectMeta.Labels[clusterv1.MachineControlPlaneLabel]
	return ok
}

// IsRunning returns if the machine is running
func IsRunning(m *infrav1beta1.MaasMachine) bool {
	if !m.Status.MachinePowered {
		return false
	}

	state := m.Status.MachineState
	return state != nil && infrav1beta1.MachineRunningStates.Has(string(*state))
}

func getExternalMachineIP(log logr.Logger, preferredSubnets []string, machine *infrav1beta1.MaasMachine) string {
	for _, i := range machine.Status.Addresses {
		if i.Type == clusterv1.MachineExternalIP && addressInPreferredSubnets(log, preferredSubnets, i.Address) {
			return i.Address
		}
	}
	return ""
}

func addressInPreferredSubnets(log logr.Logger, preferredSubnets []string, address string) bool {
	if len(preferredSubnets) == 0 {
		return true
	}

	for _, subnet := range preferredSubnets {
		_, subnetParsed, err := net.ParseCIDR(subnet)
		if err != nil {
			log.Error(err, "unable to parse subent", "subnet", subnet)
			continue
		}

		addressParsed := net.ParseIP(address)
		if subnetParsed.Contains(addressParsed) {
			return true
		} else {
			continue
		}
	}

	return false
}

func (r *MaasClusterReconciler) reconcileNormal(_ context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	clusterScope.Info("Reconciling MaasCluster")

	maasCluster := clusterScope.MaasCluster

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(maasCluster, infrav1beta1.ClusterFinalizer) {
		controllerutil.AddFinalizer(maasCluster, infrav1beta1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	dnsService := dns.NewService(clusterScope)

	if err := dnsService.ReconcileDNS(); err != nil {
		clusterScope.Error(err, "failed to reconcile load balancer")
		conditions.MarkFalse(maasCluster, infrav1beta1.DNSReadyCondition, infrav1beta1.DNSFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{}, err
	}

	if maasCluster.Status.Network.DNSName == "" {
		conditions.MarkFalse(maasCluster, infrav1beta1.DNSReadyCondition, infrav1beta1.WaitForDNSNameReason, clusterv1.ConditionSeverityInfo, "")
		clusterScope.Info("Waiting on API server DNS name")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	maasCluster.Spec.ControlPlaneEndpoint = infrav1beta1.APIEndpoint{
		Host: maasCluster.Status.Network.DNSName,
		Port: clusterScope.APIServerPort(),
	}

	maasCluster.Status.Ready = true

	// Mark the maasCluster ready
	conditions.MarkTrue(maasCluster, infrav1beta1.DNSReadyCondition)

	if err := r.reconcileDNSAttachments(clusterScope, dnsService); err != nil {
		if errors.Is(err, ErrRequeueDNS) {
			return ctrl.Result{}, nil
			//return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		clusterScope.Error(err, "failed to reconcile load balancer")
		return reconcile.Result{}, err

	}

	clusterScope.ReconcileMaasClusterWhenAPIServerIsOnline()
	if k, _ := clusterScope.IsAPIServerOnline(); !k {
		conditions.MarkFalse(maasCluster, infrav1beta1.APIServerAvailableCondition, infrav1beta1.APIServerNotReadyReason, clusterv1.ConditionSeverityWarning, "")
		return ctrl.Result{}, nil
	}

	conditions.MarkTrue(maasCluster, infrav1beta1.APIServerAvailableCondition)
	clusterScope.Info("API Server is available")

	if clusterScope.IsLXDControlPlaneCluster() {

		// Ensure LXD initializer DaemonSet exists/absent as needed
		if err := r.ensureLXDInitializerDS(context.Background(), clusterScope); err != nil {
			clusterScope.Error(err, "failed to reconcile LXD initializer DaemonSet")
			return reconcile.Result{}, err
		}

		lxdService := lxd.NewService(clusterScope)
		if err := lxdService.ReconcileLXD(); err != nil {
			clusterScope.Error(err, "failed to reconcile LXD hosts")
			conditions.MarkFalse(maasCluster, infrav1beta1.LXDReadyCondition, infrav1beta1.LXDFailedReason, clusterv1.ConditionSeverityError, err.Error())
			return reconcile.Result{}, err
		}
		conditions.MarkTrue(maasCluster, infrav1beta1.LXDReadyCondition)
		clusterScope.Info("LXD hosts are available")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager will add watches for this controller
func (r *MaasClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recover := true
	if r.GenericEventChannel == nil {
		r.GenericEventChannel = make(chan event.GenericEvent)
	}

	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MaasCluster{}).
		WithOptions(controller.Options{
			RecoverPanic: &recover,
		}).
		Watches(
			&infrav1beta1.MaasMachine{},
			handler.EnqueueRequestsFromMapFunc(r.controlPlaneMachineToCluster),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(
				util.ClusterToInfrastructureMapFunc(context.Background(), infrav1beta1.GroupVersion.WithKind("MaasCluster"), mgr.GetClient(), &infrav1beta1.MaasCluster{}),
			),
			//predicates.ClusterUnpaused(mgr.GetScheme(), r.Log),
		).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), r.Log)).
		Build(r)
	if err != nil {
		return err
	}

	if err := c.Watch(
		source.Channel(r.GenericEventChannel, &handler.EnqueueRequestForObject{}),
	); err != nil {
		return err
	}

	return err

}

// controlPlaneMachineToCluster is a handler.ToRequestsFunc to be used
// to enqueue requests for reconciliation for MaasCluster to update
// its status.apiEndpoints field.
func (r *MaasClusterReconciler) controlPlaneMachineToCluster(_ context.Context, o client.Object) []ctrl.Request {
	maasMachine, ok := o.(*infrav1beta1.MaasMachine)
	if !ok {
		r.Log.Error(nil, fmt.Sprintf("expected a MaasMachine but got a %T", o))
		return nil
	}
	if !IsControlPlaneMachine(maasMachine) {
		return nil
	}

	ctx := context.TODO()

	// Fetch the CAPI Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, maasMachine.ObjectMeta)
	if err != nil {
		r.Log.Error(err, "MaasMachine is missing cluster label or cluster does not exist",
			"namespace", maasMachine.Namespace, "name", maasMachine.Name)
		return nil
	}

	// Fetch the MaasCluster
	maasCluster := &infrav1beta1.MaasCluster{}
	maasClusterKey := client.ObjectKey{
		Namespace: maasMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, maasClusterKey, maasCluster); err != nil {
		r.Log.Error(err, "failed to get MaasCluster",
			"namespace", maasClusterKey.Namespace, "name", maasClusterKey.Name)
		return nil
	}

	return []ctrl.Request{{
		NamespacedName: types.NamespacedName{
			Namespace: maasClusterKey.Namespace,
			Name:      maasClusterKey.Name,
		},
	}}
}
