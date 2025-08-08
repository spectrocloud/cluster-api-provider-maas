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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	infrautil "github.com/spectrocloud/cluster-api-provider-maas/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
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
	if infrautil.IsCustomEndpointPresent(s.MaasCluster.GetAnnotations()) && s.MaasCluster.Spec.ControlPlaneEndpoint.Port != 0 {
		return s.MaasCluster.Spec.ControlPlaneEndpoint.Port
	}

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
	if infrautil.IsCustomEndpointPresent(s.MaasCluster.GetAnnotations()) {
		s.SetDNSName(s.MaasCluster.Spec.ControlPlaneEndpoint.Host)
		return s.MaasCluster.Spec.ControlPlaneEndpoint.Host
	}

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
	labels := map[string]string{clusterv1.ClusterNameLabel: s.Cluster.Name}

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

// GetControlPlaneMaasMachines returns all MaasMachine objects associated with control plane machines in the cluster
func (s *ClusterScope) GetControlPlaneMaasMachines() ([]*infrav1beta1.MaasMachine, error) {
	machines, err := s.GetClusterMaasMachines()
	if err != nil {
		return nil, err
	}

	var cpMachines []*infrav1beta1.MaasMachine
	for _, machine := range machines {
		// Check for control plane label
		if _, ok := machine.ObjectMeta.Labels[clusterv1.MachineControlPlaneLabel]; ok {
			cpMachines = append(cpMachines, machine)
		}
	}

	return cpMachines, nil
}

// SetStatus sets the MaasCluster status
func (s *ClusterScope) SetStatus(status infrav1beta1.MaasClusterStatus) {
	s.MaasCluster.Status = status
}

// GetMaasClientIdentity returns the MAAS client identity
func (s *ClusterScope) GetMaasClientIdentity() ClientIdentity {
	// Try to get MAAS credentials from a secret
	// The secret is expected to be in the same namespace as the MaasCluster
	// and named "maas-credentials" by default
	// Secret containing MAAS endpoint/token created by Palette bootstrapper
	// Default name switched from "maas-credentials" to "capmaas-manager-bootstrap-credentials"
	secretName := "capmaas-manager-bootstrap-credentials"

	// Get the secret
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: s.MaasCluster.Namespace,
		Name:      secretName,
	}

	// Try to get the secret
	err := s.client.Get(context.Background(), key, secret)
	if err != nil {
		// If the secret doesn't exist, fall back to environment variables or default values
		s.Info("Failed to get MAAS bootstrap credentials secret, using fallback values", "error", err)
		return ClientIdentity{
			URL:   getEnvOrDefault("MAAS_API_URL", "http://localhost:5240/MAAS"),
			Token: getEnvOrDefault("MAAS_API_TOKEN", "dummy-token"),
		}
	}

	// Get the credentials from the secret
	url := string(secret.Data["url"])
	token := string(secret.Data["token"])

	// Validate the credentials
	if url == "" || token == "" {
		s.Info("Invalid MAAS credentials in secret, using fallback values")
		return ClientIdentity{
			URL:   getEnvOrDefault("MAAS_API_URL", "http://localhost:5240/MAAS"),
			Token: getEnvOrDefault("MAAS_API_TOKEN", "dummy-token"),
		}
	}

	return ClientIdentity{
		URL:   url,
		Token: token,
	}
}

// getEnvOrDefault returns the value of the environment variable or the default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

// ClientIdentity contains MAAS client identity information
type ClientIdentity struct {
	URL   string
	Token string
}

const (
	maasPreferredSubnetConfigmap = "maas-preferred-subnet"
	preferredSubnetKey           = "preferredSubnets"
)

func (s *ClusterScope) GetPreferredSubnets() ([]string, error) {
	maasPreferredSubnet := &corev1.ConfigMap{}
	err := s.client.Get(context.Background(), types.NamespacedName{
		Namespace: s.Cluster.GetNamespace(),
		Name:      maasPreferredSubnetConfigmap,
	}, maasPreferredSubnet)
	switch {
	case err != nil && !apierrors.IsNotFound(err):
		return nil, err
	case err != nil && apierrors.IsNotFound(err):
		return nil, nil
	}

	subnetsString := maasPreferredSubnet.Data[preferredSubnetKey]
	var result []string
	for _, subnet := range strings.Split(subnetsString, ",") {
		result = append(result, strings.TrimSpace(subnet))
	}
	return result, nil
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

	ctx := context.TODO()

	cluster := &clusterv1.Cluster{}
	if err := s.client.Get(ctx, util.ObjectKey(s.Cluster), cluster); err != nil {
		return false, err
	} else if !cluster.DeletionTimestamp.IsZero() {
		s.Info("Cluster is deleting; abort APIServerOnline check", "cluster", cluster.Name)
		return false, errors.New("Cluster is deleting; abort IsAPIServerOnline")
	}

	remoteClient, err := s.tracker.GetClient(ctx, util.ObjectKey(s.Cluster))
	if err != nil {
		s.V(2).Info("Waiting for online server to come online")
		return false, nil
	}

	err = remoteClient.List(ctx, new(corev1.NodeList))

	return err == nil, nil
}

// IsCustomEndpoint returns true if the cluster has a custom endpoint
func (s *ClusterScope) IsCustomEndpoint() bool {
	if infrautil.IsCustomEndpointPresent(s.MaasCluster.GetAnnotations()) {
		s.GetDNSName()
		s.V(0).Info("custom dns is provided skipping dns reconcile", "dns", s.GetDNSName())
		return true
	}
	return false
}

// IsLXDControlPlaneCluster returns true if LXD control plane cluster is enabled
func (s *ClusterScope) IsLXDControlPlaneCluster() bool {
	if s.MaasCluster.Spec.LXDControlPlaneCluster == nil {
		return false
	}
	return *s.MaasCluster.Spec.LXDControlPlaneCluster
}

// GetLXDConfig returns the LXD configuration
func (s *ClusterScope) GetLXDConfig() *infrav1beta1.LXDConfig {
	return s.MaasCluster.Spec.LXDConfig
}

// IsWorkloadCluster returns true if this is a workload cluster
func (s *ClusterScope) IsWorkloadCluster() bool {
	return s.MaasCluster.Spec.InfrastructureClusterRef != nil
}

// GetInfrastructureClusterRef returns the infrastructure cluster reference
func (s *ClusterScope) GetInfrastructureClusterRef() *infrav1beta1.InfrastructureClusterRef {
	return s.MaasCluster.Spec.InfrastructureClusterRef
}

// GetWorkloadClusterConfig returns the workload cluster configuration
func (s *ClusterScope) GetWorkloadClusterConfig() *infrav1beta1.WorkloadClusterConfig {
	return s.MaasCluster.Spec.WorkloadClusterConfig
}

// GetInfrastructureCluster retrieves the infrastructure cluster
func (s *ClusterScope) GetInfrastructureCluster() (*infrav1beta1.MaasCluster, error) {
	if !s.IsWorkloadCluster() {
		return nil, errors.New("not a workload cluster")
	}

	ref := s.GetInfrastructureClusterRef()
	if ref == nil {
		return nil, errors.New("infrastructure cluster reference is nil")
	}

	namespace := ref.Namespace
	if namespace == "" {
		namespace = s.MaasCluster.Namespace
	}

	infraCluster := &infrav1beta1.MaasCluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      ref.Name,
	}

	if err := s.client.Get(context.TODO(), key, infraCluster); err != nil {
		return nil, errors.Wrapf(err, "failed to get infrastructure cluster %s/%s", namespace, ref.Name)
	}

	return infraCluster, nil
}

// GetNodePoolConfig returns the node pool configuration for a given machine
func (s *ClusterScope) GetNodePoolConfig(machine *clusterv1.Machine) *infrav1beta1.NodePoolConfig {
	if !s.IsWorkloadCluster() {
		return nil
	}

	config := s.GetWorkloadClusterConfig()
	if config == nil {
		return nil
	}

	// Check if it's a control plane machine
	if util.IsControlPlaneMachine(machine) {
		return config.ControlPlanePool
	}

	// Check worker pools
	for _, pool := range config.WorkerPools {
		// This is a simplified check - in real implementation, you'd match based on
		// machine labels, annotations, or other criteria
		if pool.Name == machine.Labels["node-pool"] {
			return &pool
		}
	}

	return nil
}

// ShouldUseLXDForMachine determines if a machine should use LXD based on node pool configuration
func (s *ClusterScope) ShouldUseLXDForMachine(machine *clusterv1.Machine) bool {
	// Check machine-specific dynamicLXD flag first
	if machine.Spec.InfrastructureRef.Kind == "MaasMachine" {
		// This would require fetching the MaasMachine to check its spec
		// For now, we'll rely on the node pool configuration
	}

	// Check node pool configuration
	poolConfig := s.GetNodePoolConfig(machine)
	if poolConfig != nil && poolConfig.UseLXD != nil {
		return *poolConfig.UseLXD
	}

	// Default to false for backward compatibility
	return false
}

// GetStaticIPForMachine returns the static IP for a machine based on node pool configuration
func (s *ClusterScope) GetStaticIPForMachine(machine *clusterv1.Machine) string {
	poolConfig := s.GetNodePoolConfig(machine)
	if poolConfig == nil || len(poolConfig.StaticIPs) == 0 {
		return ""
	}

	// Simple round-robin assignment - in real implementation, you'd want more sophisticated logic
	maasMachines, err := s.GetClusterMaasMachines()
	if err != nil {
		s.Error(err, "failed to get cluster machines for static IP assignment")
		return ""
	}
	index := len(maasMachines) % len(poolConfig.StaticIPs)
	return poolConfig.StaticIPs[index]
}

// GetMappedAvailabilityZone maps workload cluster AZ to infrastructure cluster AZ
func (s *ClusterScope) GetMappedAvailabilityZone(workloadAZ string) string {
	if !s.IsWorkloadCluster() {
		return workloadAZ
	}

	config := s.GetWorkloadClusterConfig()
	if config == nil || config.AZMapping == nil {
		return workloadAZ
	}

	if mappedAZ, exists := config.AZMapping[workloadAZ]; exists {
		return mappedAZ
	}

	return workloadAZ
}

// GetMappedResourcePool maps workload cluster resource pool to infrastructure cluster resource pool
func (s *ClusterScope) GetMappedResourcePool(workloadPool string) string {
	if !s.IsWorkloadCluster() {
		return workloadPool
	}

	config := s.GetWorkloadClusterConfig()
	if config == nil || config.ResourcePoolMapping == nil {
		return workloadPool
	}

	if mappedPool, exists := config.ResourcePoolMapping[workloadPool]; exists {
		return mappedPool
	}

	return workloadPool
}
