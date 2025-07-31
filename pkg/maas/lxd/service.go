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
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

// Service provides LXD operations for a cluster
type Service struct {
	clusterScope *scope.ClusterScope
}

// NewService creates a new LXD service
func NewService(clusterScope *scope.ClusterScope) *Service {
	return &Service{
		clusterScope: clusterScope,
	}
}

// ReconcileLXD reconciles LXD setup for a cluster
func (s *Service) ReconcileLXD() error {
	// Check if this is a control plane cluster that should use LXD
	if !s.clusterScope.IsLXDControlPlaneCluster() {
		return nil
	}

	// Get the cluster
	cluster := s.clusterScope.MaasCluster

	// // Check if LXD is already ready
	// if conditions.IsTrue(cluster, v1beta1.LXDReadyCondition) {
	// 	return nil
	// }

	// Even if LXDReady is already true we still verify that each control-plane node remains registered.

	// Set the LXD setup pending condition
	conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "LXD setup is pending")

	// Get the control plane machines
	cpMachines, err := s.clusterScope.GetControlPlaneMaasMachines()
	if err != nil {
		conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDFailedReason, clusterv1.ConditionSeverityError, "Failed to get control plane machines: %v", err)
		return errors.Wrap(err, "failed to get control plane machines")
	}

	// Check if there are any control plane machines
	if len(cpMachines) == 0 {
		conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "No control plane machines found")
		return nil
	}

	// Check if all control plane machines are ready
	for _, machine := range cpMachines {
		if !machine.Status.Ready {
			conditions.MarkFalse(cluster, v1beta1.LXDReadyCondition, v1beta1.LXDSetupPendingReason, clusterv1.ConditionSeverityInfo, "Control plane machine %s is not ready", machine.Name)
			return nil
		}
	}

	// Set up LXD on each control plane machine
	for _, machine := range cpMachines {
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
