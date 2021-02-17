package scope

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client         client.Client
	Logger         logr.Logger
	Cluster        *clusterv1.Cluster
	MaasCluster    *infrav1.MaasCluster
	ControllerName string
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster        *clusterv1.Cluster
	MaasCluster    *infrav1.MaasCluster
	controllerName string
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	//session, serviceLimiters, err := sessionForRegion(params.MaasCluster.Spec.Region, params.Endpoints)
	//if err != nil {
	//	return nil, errors.Errorf("failed to create maas session: %v", err)
	//}

	helper, err := patch.NewHelper(params.MaasCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &ClusterScope{
		Logger:         params.Logger,
		client:         params.Client,
		Cluster:        params.Cluster,
		MaasCluster:    params.MaasCluster,
		patchHelper:    helper,
		controllerName: params.ControllerName,
	}, nil
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process (instead we are hiding it during the deletion process).
	conditions.SetSummary(s.MaasCluster,
		conditions.WithConditions(
			infrav1.DNSReadyCondition,
		),
		conditions.WithStepCounterIf(s.MaasCluster.ObjectMeta.DeletionTimestamp.IsZero()),
	)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return s.patchHelper.Patch(
		context.TODO(),
		s.MaasCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.DNSReadyCondition,
		}},
	)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}

// APIServerPort returns the APIServerPort to use when creating the load balancer.
func (s *ClusterScope) APIServerPort() int {
	if s.Cluster.Spec.ClusterNetwork != nil && s.Cluster.Spec.ClusterNetwork.APIServerPort != nil {
		return int(*s.Cluster.Spec.ClusterNetwork.APIServerPort)
	}
	return 6443
}

// SetDNSName sets the Network systemID in spec.
func (s *ClusterScope) SetDNSName(dnsName string) {
	s.MaasCluster.Status.Network.DNSName = dnsName
}

// GetDNSName sets the Network systemID in spec.
// This can't do a lookup on Status.Network.DNSDomain name since it's derviced from here
func (s *ClusterScope) GetDNSName() string {
	return fmt.Sprintf("%s.%s", s.Cluster.Name, s.MaasCluster.Spec.DNSDomain)
}

// GetActiveMaasMachines all MaaS machines NOT being deleted
func (s *ClusterScope) GetClusterMaasMachines() ([]*infrav1.MaasMachine, error) {

	machineList := &infrav1.MaasMachineList{}
	labels := map[string]string{clusterv1.ClusterLabelName: s.Cluster.Name}

	if err := s.client.List(
		context.TODO(),
		machineList,
		client.InNamespace(s.Cluster.Namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, errors.Wrap(err, "failed to list machines")
	}

	var machines []*infrav1.MaasMachine
	for i := range machineList.Items {
		m := &machineList.Items[i]
		machines = append(machines, m)
		// TODO need active?
		//if m.DeletionTimestamp.IsZero() {
		//}
	}

	return machines, nil
}
