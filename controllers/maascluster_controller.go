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
	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/dns"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
)

// MaasClusterReconciler reconciles a MaasCluster object
type MaasClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=maasclusters/status,verbs=get;update;patch

// Reconcile reads that state of the cluster for a MaasCluster object and makes changes based on the state read
// and what is in the MaasCluster.Spec
func (r *MaasClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("maascluster", req.NamespacedName)

	// Fetch the MaasCluster instance
	maasCluster := &infrav1.MaasCluster{}
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

	log = log.WithValues("cluster", cluster.Name)

	//// Create a helper for managing a maas container hosting the loadbalancer.
	//externalLoadBalancer, err := maas.NewLoadBalancer(cluster.Name)
	//if err != nil {
	//	return ctrl.Result{}, errors.Wrapf(err, "failed to create helper for managing the externalLoadBalancer")
	//}

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:         r.Client,
		Logger:         log,
		Cluster:        cluster,
		MaasCluster:    maasCluster,
		ControllerName: "maascluster",
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AWSCluster changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Support FailureDomains
	// In cloud providers this would likely look up which failure domains are supported and set the status appropriately.
	// In the case of Maas, failure domains don't mean much so we simply copy the Spec into the Status.
	maasCluster.Status.FailureDomains = maasCluster.Spec.FailureDomains

	// Handle deleted clusters
	if !maasCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, clusterScope)
}

func (r *MaasClusterReconciler) reconcileDelete(_ context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	clusterScope.Info("Reconciling MaasCluster delete")

	maasCluster := clusterScope.MaasCluster

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(maasCluster, infrav1.ClusterFinalizer)

	// TODO(saamalik) implement the recorder stuff (look at aws)

	return reconcile.Result{}, nil
}

func (r *MaasClusterReconciler) reconcileNormal(_ context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	clusterScope.Info("Reconciling MaasCluster")

	maasCluster := clusterScope.MaasCluster

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(maasCluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(maasCluster, infrav1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	dnsService := dns.NewService(clusterScope)

	if err := dnsService.ReconcileLoadbalancers(); err != nil {
		clusterScope.Error(err, "failed to reconcile load balancer")
		conditions.MarkFalse(maasCluster, infrav1.LoadBalancerReadyCondition, infrav1.LoadBalancerFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{}, err
	}

	if maasCluster.Status.Network.DNSName == "" {
		conditions.MarkFalse(maasCluster, infrav1.LoadBalancerReadyCondition, infrav1.WaitForDNSNameReason, clusterv1.ConditionSeverityInfo, "")
		clusterScope.Info("Waiting on API server DNS name")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	// Mark the maasCluster ready
	conditions.MarkTrue(maasCluster, infrav1.LoadBalancerReadyCondition)

	maasCluster.Spec.ControlPlaneEndpoint = infrav1.APIEndpoint{
		Host: maasCluster.Status.Network.DNSName,
		Port: clusterScope.APIServerPort(),
	}

	maasCluster.Status.Ready = true
	return ctrl.Result{}, nil
}

// SetupWithManager will add watches for this controller
func (r *MaasClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.MaasCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(r.Log)).
		Build(r)
	if err != nil {
		return err
	}
	return c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("MaasCluster"))),
		predicates.ClusterUnpaused(r.Log),
	)
}
