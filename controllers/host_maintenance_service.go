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
	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// HostEvacuationFinalizer is the finalizer that blocks deletion until evacuation criteria are met
	HostEvacuationFinalizer = "maas.lxd.io/host-evacuation"

	// EvacuationCheckInterval is the interval for checking evacuation gates
	EvacuationCheckInterval = 10 * time.Second
)

// HostMaintenanceService provides evacuation logic for MaasMachine controller
type HostMaintenanceService struct {
	client           client.Client
	namespace        string
	tagService       maint.TagService
	inventoryService maint.InventoryService
}

// NewHostMaintenanceService creates a new host maintenance service
func NewHostMaintenanceService(k8sClient client.Client, namespace string) *HostMaintenanceService {
	maasClient := maint.NewMAASClient(k8sClient, namespace)
	return &HostMaintenanceService{
		client:           k8sClient,
		namespace:        namespace,
		tagService:       maint.NewTagService(maasClient),
		inventoryService: maint.NewInventoryService(maasClient),
	}
}

// AddEvacuationFinalizer adds the evacuation finalizer to a host machine
func (s *HostMaintenanceService) AddEvacuationFinalizer(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) error {
	// Only add finalizer to host machines (not VMs)
	if !s.isHostMachine(maasMachine) {
		return nil
	}

	if !containsString(maasMachine.Finalizers, HostEvacuationFinalizer) {
		log.Info("Adding host evacuation finalizer")
		maasMachine.Finalizers = append(maasMachine.Finalizers, HostEvacuationFinalizer)
		return s.client.Update(ctx, maasMachine)
	}
	return nil
}

// CheckEvacuationGates checks if evacuation criteria are met for host deletion
func (s *HostMaintenanceService) CheckEvacuationGates(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) (bool, error) {
	if maasMachine.Spec.SystemID == nil {
		return false, fmt.Errorf("maasMachine has no systemID")
	}
	hostSystemID := *maasMachine.Spec.SystemID

	// Gate 1: Check if host is empty (no VMs running)
	hostEmpty, err := s.isHostEmpty(ctx, hostSystemID, log)
	if err != nil {
		return false, fmt.Errorf("failed to check if host is empty: %w", err)
	}

	if !hostEmpty {
		log.Info("Host not empty, evacuation blocked", "host", hostSystemID)
		return false, nil
	}

	// Gate 2: Check per-WLC ready-op-<uuid> tags on CP VMs
	wlcReady, err := s.checkWLCReadyTags(ctx, hostSystemID, log)
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

// ClearMaintenanceTagsAndRemoveFinalizer clears maintenance tags and removes the evacuation finalizer
func (s *HostMaintenanceService) ClearMaintenanceTagsAndRemoveFinalizer(ctx context.Context, maasMachine *infrav1beta1.MaasMachine, log logr.Logger) error {
	if maasMachine.Spec.SystemID == nil {
		return fmt.Errorf("maasMachine has no systemID")
	}
	hostSystemID := *maasMachine.Spec.SystemID

	// Clear maintenance tags
	if err := s.clearMaintenanceTags(ctx, hostSystemID, log); err != nil {
		log.Error(err, "failed to clear maintenance tags")
		return err
	}

	// Remove finalizer
	log.Info("Removing host evacuation finalizer")
	maasMachine.Finalizers = removeString(maasMachine.Finalizers, HostEvacuationFinalizer)
	if err := s.client.Update(ctx, maasMachine); err != nil {
		log.Error(err, "failed to remove finalizer")
		return err
	}

	log.Info("Host evacuation completed successfully")
	return nil
}

// isHostMachine checks if the MaasMachine is a host (not a VM)
func (s *HostMaintenanceService) isHostMachine(maasMachine *infrav1beta1.MaasMachine) bool {
	// Check if this is a host machine (not a VM)
	// VMs typically have a parent reference, hosts don't
	return maasMachine.Spec.Parent == nil || *maasMachine.Spec.Parent == ""
}

// isHostEmpty checks if the host has no running VMs
func (s *HostMaintenanceService) isHostEmpty(ctx context.Context, hostSystemID string, log logr.Logger) (bool, error) {
	vms, err := s.inventoryService.ListHostVMs(hostSystemID)
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
func (s *HostMaintenanceService) checkWLCReadyTags(ctx context.Context, hostSystemID string, log logr.Logger) (bool, error) {
	// Get all VMs on the host
	vms, err := s.inventoryService.ListHostVMs(hostSystemID)
	if err != nil {
		return false, fmt.Errorf("failed to list host VMs: %w", err)
	}

	// Check each VM for ready-op-<uuid> tags
	for _, vm := range vms {
		vmDetails, err := s.inventoryService.GetVM(vm.SystemID)
		if err != nil {
			log.Error(err, "failed to get VM details", "vm", vm.SystemID)
			continue
		}

		// Check if VM has any ready-op-<uuid> tags
		hasReadyTag := false
		for _, tag := range vmDetails.Tags {
			if len(tag) > len(maint.TagVMReadyOpPrefix) && tag[:len(maint.TagVMReadyOpPrefix)] == maint.TagVMReadyOpPrefix {
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
func (s *HostMaintenanceService) clearMaintenanceTags(ctx context.Context, hostSystemID string, log logr.Logger) error {
	// Get current machine details to check existing tags
	hostDetails, err := s.inventoryService.GetHost(hostSystemID)
	if err != nil {
		log.Error(err, "failed to get host details for tag cleanup", "host", hostSystemID)
		return err
	}

	// Clear static maintenance tags
	maintenanceTags := []string{
		maint.TagHostMaintenance,
		maint.TagHostNoSchedule,
	}

	for _, tag := range maintenanceTags {
		if err := s.tagService.RemoveTagFromHost(hostSystemID, tag); err != nil {
			log.Error(err, "failed to remove maintenance tag", "tag", tag, "host", hostSystemID)
			return err
		}
		log.Info("Cleared maintenance tag", "tag", tag, "host", hostSystemID)
	}

	// Clear dynamic operation ID tags (maas.lxd-hcp-op-*)
	for _, tag := range hostDetails.Tags {
		if len(tag) > len(maint.TagHostOpPrefix) && tag[:len(maint.TagHostOpPrefix)] == maint.TagHostOpPrefix {
			if err := s.tagService.RemoveTagFromHost(hostSystemID, tag); err != nil {
				log.Error(err, "failed to remove operation ID tag", "tag", tag, "host", hostSystemID)
				return err
			}
			log.Info("Cleared operation ID tag", "tag", tag, "host", hostSystemID)
		}
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
