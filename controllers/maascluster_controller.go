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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
)

// MaasClusterReconciler reconciles a MaasCluster object
type MaasClusterReconciler struct {
	client.Client
	Log logr.Logger
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

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(maasCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the MaasCluster object and status after each reconciliation.
	defer func() {
		if err := patchMaasCluster(ctx, patchHelper, maasCluster); err != nil {
			log.Error(err, "failed to patch MaasCluster")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Support FailureDomains
	// In cloud providers this would likely look up which failure domains are supported and set the status appropriately.
	// In the case of Maas, failure domains don't mean much so we simply copy the Spec into the Status.
	maasCluster.Status.FailureDomains = maasCluster.Spec.FailureDomains

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(maasCluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(maasCluster, infrav1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deleted clusters
	if !maasCluster.DeletionTimestamp.IsZero() {
		log.Info("need to implement deletion")
		return ctrl.Result{}, nil
		//return r.reconcileDelete(ctx, maasCluster, externalLoadBalancer)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, maasCluster, externalLoadBalancer)
}

func patchMaasCluster(ctx context.Context, patchHelper *patch.Helper, maasCluster *infrav1.MaasCluster) error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process (instead we are hiding it during the deletion process).
	conditions.SetSummary(maasCluster,
		conditions.WithConditions(
			infrav1.LoadBalancerAvailableCondition,
		),
		conditions.WithStepCounterIf(maasCluster.ObjectMeta.DeletionTimestamp.IsZero()),
	)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return patchHelper.Patch(
		ctx,
		maasCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.LoadBalancerAvailableCondition,
		}},
	)
}

func (r *MaasClusterReconciler) reconcileNormal(ctx context.Context, maasCluster *infrav1.MaasCluster, externalLoadBalancer *maas.LoadBalancer) (ctrl.Result, error) {
	//Create the maas container hosting the load balancer
	if err := externalLoadBalancer.Create(ctx); err != nil {
		conditions.MarkFalse(maasCluster, infrav1.LoadBalancerAvailableCondition, infrav1.LoadBalancerProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return ctrl.Result{}, errors.Wrap(err, "failed to create load balancer")
	}

	// Set APIEndpoints with the load balancer IP so the Cluster API Cluster Controller can pull it
	lbip4, err := externalLoadBalancer.IP(ctx)
	if err != nil {
		conditions.MarkFalse(maasCluster, infrav1.LoadBalancerAvailableCondition, infrav1.LoadBalancerProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return ctrl.Result{}, errors.Wrap(err, "failed to get ip for the load balancer")
	}

	maasCluster.Spec.ControlPlaneEndpoint = infrav1.APIEndpoint{
		Host: lbip4,
		Port: 6443,
	}

	// Mark the maasCluster ready
	maasCluster.Status.Ready = true
	conditions.MarkTrue(maasCluster, infrav1.LoadBalancerAvailableCondition)

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
