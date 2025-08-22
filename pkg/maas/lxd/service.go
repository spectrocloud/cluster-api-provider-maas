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

package lxd

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service provides LXD operations for a cluster
type Service struct {
	clusterScope *scope.ClusterScope
}

const (
	// LXD Host initialization label
	LXDHostInitializedLabel = "lxdhost.cluster.com/initialized"
)

// NewService creates a new LXD service
func NewService(clusterScope *scope.ClusterScope) *Service {
	return &Service{
		clusterScope: clusterScope,
	}
}

// ReconcileLXD reconciles LXD setup for a cluster
func (s *Service) ReconcileLXD() error {
	// Check if this is a control plane cluster that should use LXD
	if !s.clusterScope.IsLXDHostEnabled() {
		return nil
	}

	// Get the cluster
	cluster := s.clusterScope.MaasCluster

	// Set the LXD setup pending condition
	conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "LXD setup is pending")

	// Get ALL machines in the cluster (control plane + worker nodes)
	allMachines, err := s.clusterScope.GetClusterMaasMachines()
	if err != nil {
		conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDFailedReason, clusterv1.ConditionSeverityError, "Failed to get cluster machines: %v", err)
		return errors.Wrap(err, "failed to get cluster machines")
	}

	// Check if there are any machines
	if len(allMachines) == 0 {
		conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "No machines found in cluster")
		return nil
	}

	// Check if all machines are ready
	for _, machine := range allMachines {
		if !machine.Status.Ready {
			conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "Machine %s is not ready", machine.Name)
			return nil
		}
	}

	// Set up LXD on each machine
	for _, machine := range allMachines {
		if err := s.setupLXDOnMachine(machine); err != nil {
			conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDFailedReason, clusterv1.ConditionSeverityError, "Failed to set up LXD on machine %s: %v", machine.Name, err)
			return errors.Wrapf(err, "failed to set up LXD on machine %s", machine.Name)
		}
	}

	// Mark LXD as ready
	conditions.MarkTrue(cluster, v1beta1.LXDReadyCondition)

	// Update the cluster status
	s.clusterScope.SetStatus(cluster.Status)

	return nil
}

// setupLXDOnMachine sets up LXD on a machine
func (s *Service) setupLXDOnMachine(machine *v1beta1.MaasMachine) error {
	// Select the first valid IP (preferring ExternalIP, then InternalIP)
	var nodeIP string
	for _, addr := range machine.Status.Addresses {
		if net.ParseIP(addr.Address) == nil {
			continue // skip hostnames
		}
		if addr.Type == clusterv1.MachineExternalIP {
			nodeIP = addr.Address
			break
		}
		if addr.Type == clusterv1.MachineInternalIP && nodeIP == "" {
			nodeIP = addr.Address
		}
	}
	if nodeIP == "" {
		return fmt.Errorf("machine %s has no valid IP address", machine.Name)
	}

	s.clusterScope.Info("Setting up LXD host", "machine", machine.Name, "ip", nodeIP)

	// Get the LXD configuration from the cluster
	lxdConfig := s.clusterScope.GetLXDConfig()

	// Create the host configuration
	hostConfig := HostConfig{
		NodeIP:          nodeIP,
		MaasAPIKey:      s.clusterScope.GetMaasClientIdentity().Token,
		MaasAPIEndpoint: s.clusterScope.GetMaasClientIdentity().URL,
		StorageBackend:  lxdConfig.StorageBackend,
		StorageSize:     lxdConfig.StorageSize,
		NetworkBridge:   lxdConfig.NetworkBridge,
		ResourcePool:    lxdConfig.ResourcePool,
		Zone:            lxdConfig.Zone,
		TrustPassword:   "capmaas",
	}

	// Check if LXD initialization is complete on the node before attempting MAAS registration
	lxdReady, err := s.isNodeLXDInitialized(machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check LXD initialization status for machine %s", machine.Name)
	}

	if !lxdReady {
		s.clusterScope.V(1).Info("LXD not yet initialized, will retry", "machine", machine.Name)
		// Return a specific error to trigger requeue instead of silently skipping
		return fmt.Errorf("LXD not ready on machine %s, waiting for initialization", machine.Name)
	}

	// Set up LXD on the machine
	// Note: This now relies on the DaemonSet to initialize LXD
	// It only checks if the host is registered with MAAS and registers it if not
	// Use our adapter implementation to ensure compatibility with MAAS 3.x
	// Use the maas-client-go implementation (Zone/Pool struct fix in PR #19)
	if err := SetupLXDHostWithMaasClient(hostConfig); err != nil {
		return errors.Wrapf(err, "failed to set up LXD on machine %s", machine.Name)
	}

	return nil
}

// isNodeLXDInitialized checks if a node has the LXD initialization label
// The lxd-initializer DaemonSet runs on target cluster nodes and labels them when LXD init completes
func (s *Service) isNodeLXDInitialized(machine *v1beta1.MaasMachine) (bool, error) {
	// Get the node name from the machine hostname (same logic as SetNodeProviderID)
	nodeName := strings.ToLower(*machine.Status.Hostname)
	if machine.Status.Hostname == nil || *machine.Status.Hostname == "" {
		return false, fmt.Errorf("machine %s has no hostname set", machine.Name)
	}

	// Get the workload cluster client to check node labels in target cluster
	ctx := context.Background()
	remoteClient, err := s.clusterScope.GetWorkloadClusterClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get workload cluster client: %w", err)
	}

	// Get the node from target cluster
	node := &corev1.Node{}
	if err := remoteClient.Get(ctx, client.ObjectKey{Name: nodeName}, node); err != nil {
		return false, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// Check if the node has the LXD initialization label set by lxd-initializer
	if node.Labels == nil {
		return false, nil
	}

	value, exists := node.Labels[LXDHostInitializedLabel]
	return exists && value == "true", nil
}