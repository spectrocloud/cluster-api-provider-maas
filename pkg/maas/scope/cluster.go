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

package scope

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sync"
	"time"
)

const (
	DnsSuffixLength = 6
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client              client.Client
	Logger              logr.Logger
	Cluster             *clusterv1.Cluster
	MaasCluster         *infrav1beta1.MaasCluster
	ControllerName      string
	Tracker             *remote.ClusterCacheTracker
	ClusterEventChannel chan event.GenericEvent
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster             *clusterv1.Cluster
	MaasCluster         *infrav1beta1.MaasCluster
	controllerName      string
	tracker             *remote.ClusterCacheTracker
	clusterEventChannel chan event.GenericEvent
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {

	helper, err := patch.NewHelper(params.MaasCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &ClusterScope{
		Logger:              params.Logger,
		client:              params.Client,
		Cluster:             params.Cluster,
		MaasCluster:         params.MaasCluster,
		patchHelper:         helper,
		controllerName:      params.ControllerName,
		tracker:             params.Tracker,
		clusterEventChannel: params.ClusterEventChannel,
	}, nil
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process (instead we are hiding it during the deletion process).
	conditions.SetSummary(s.MaasCluster,
		conditions.WithConditions(
			infrav1beta1.DNSReadyCondition,
			infrav1beta1.APIServerAvailableCondition,
		),
		conditions.WithStepCounterIf(s.MaasCluster.ObjectMeta.DeletionTimestamp.IsZero()),
	)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return s.patchHelper.Patch(
		context.TODO(),
		s.MaasCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1beta1.DNSReadyCondition,
			infrav1beta1.APIServerAvailableCondition,
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
	//TODO: PCP-22 set 10.11.130.165 if it donent work using fixed ControlPlaneEndpoint
	s.MaasCluster.Status.Network.DNSName = dnsName
}

// GetDNSName sets the Network systemID in spec.
// This can't do a lookup on Status.Network.DNSDomain name since it's derviced from here
func (s *ClusterScope) GetDNSName() string {
	if !s.Cluster.Spec.ControlPlaneEndpoint.IsZero() {
		return s.Cluster.Spec.ControlPlaneEndpoint.Host
	}

	if s.MaasCluster.Status.Network.DNSName != "" {
		return s.MaasCluster.Status.Network.DNSName
	}

	uid := uuid.New().String()
	dnsName := fmt.Sprintf("%s-%s.%s", s.Cluster.Name, uid[len(uid)-DnsSuffixLength:], s.MaasCluster.Spec.DNSDomain)

	s.SetDNSName(dnsName)
	return dnsName
}

// GetActiveMaasMachines all MaaS machines NOT being deleted
func (s *ClusterScope) GetClusterMaasMachines() ([]*infrav1beta1.MaasMachine, error) {

	machineList := &infrav1beta1.MaasMachineList{}
	labels := map[string]string{clusterv1.ClusterLabelName: s.Cluster.Name}

	if err := s.client.List(
		context.TODO(),
		machineList,
		client.InNamespace(s.Cluster.Namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, errors.Wrap(err, "failed to list machines")
	}

	var machines []*infrav1beta1.MaasMachine
	for i := range machineList.Items {
		m := &machineList.Items[i]
		machines = append(machines, m)
		// TODO need active?
		//if m.DeletionTimestamp.IsZero() {
		//}
	}

	return machines, nil
}

var (
	// apiServerTriggers is used to prevent multiple goroutines for a single
	// Cluster that poll to see if the target API server is online.
	apiServerTriggers   = map[types.UID]struct{}{}
	apiServerTriggersMu sync.Mutex
)

func (s *ClusterScope) ReconcileMaasClusterWhenAPIServerIsOnline() {
	if s.Cluster.Status.ControlPlaneReady {
		s.Info("skipping reconcile when API server is online",
			"reason", "ControlPlaneReady")
		return
	} else if !s.Cluster.DeletionTimestamp.IsZero() {
		s.Info("skipping reconcile when API server is online",
			"reason", "controlPlaneDeleting")
		return
	}

	apiServerTriggersMu.Lock()
	defer apiServerTriggersMu.Unlock()
	if _, ok := apiServerTriggers[s.Cluster.UID]; ok {
		s.Info("skipping reconcile when API server is online",
			"reason", "alreadyPolling")
		return
	}
	apiServerTriggers[s.Cluster.UID] = struct{}{}
	go func() {
		// Block until the target API server is online.

		s.Info("start polling API server for online check")
		_ = wait.PollImmediateInfinite(time.Second*1, func() (bool, error) { return s.IsAPIServerOnline() })
		s.Info("stop polling API server for online check")
		s.Info("triggering GenericEvent", "reason", "api-server-online")
		s.clusterEventChannel <- event.GenericEvent{
			Object: s.MaasCluster,
		}

		apiServerTriggersMu.Lock()
		delete(apiServerTriggers, s.Cluster.UID)
		apiServerTriggersMu.Unlock()

		//// Once the control plane has been marked as initialized it is safe to
		//// remove the key from the map that prevents multiple goroutines from
		//// polling the API server to see if it is online.
		//s.Info("start polling for control plane initialized")
		//wait.PollImmediateInfinite(time.Second*1, func() (bool, error) { return r.isControlPlaneInitialized(ctx), nil }) // nolint:errcheck
		//s.Info("stop polling for control plane initialized")
	}()
}

func (s *ClusterScope) IsAPIServerOnline() (bool, error) {

	return true, nil
	//ctx := context.TODO()
	//
	//cluster := &clusterv1.Cluster{}
	//if err := s.client.Get(ctx, util.ObjectKey(s.Cluster), cluster); err != nil {
	//	return false, err
	//} else if !cluster.DeletionTimestamp.IsZero() {
	//	s.Info("Cluster is deleting; abort APIServerOnline check", "cluster", cluster.Name)
	//	return false, errors.New("Cluster is deleting; abort IsAPIServerOnline")
	//}
	//
	//remoteClient, err := s.tracker.GetClient(ctx, util.ObjectKey(s.Cluster))
	//if err != nil {
	//	s.V(2).Info("Waiting for online server to come online")
	//	return false, nil
	//}
	//
	//err = remoteClient.List(ctx, new(v1.NodeList))
	//
	//return err == nil, nil
}
