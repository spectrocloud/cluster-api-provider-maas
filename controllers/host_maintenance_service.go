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

	"github.com/go-logr/logr"
	maint "github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/maintenance"
	"github.com/spectrocloud/maas-client-go/maasclient"
)

// HostMaintenanceService provides evacuation gate checks and clearing helpers.
type HostMaintenanceService struct {
	inv  maint.InventoryService
	tags maint.TagService
	maas maasclient.ClientSetInterface
}

func NewHostMaintenanceService(inv maint.InventoryService, tags maint.TagService, maas maasclient.ClientSetInterface) *HostMaintenanceService {
	return &HostMaintenanceService{inv: inv, tags: tags, maas: maas}
}

// CheckEvacuationGates returns true when either host is empty OR all affected clusters
// have replacement CP VMs ready on different hosts with ready-op tag.
func (s *HostMaintenanceService) CheckEvacuationGates(ctx context.Context, hostSystemID, opID string, log logr.Logger) (bool, []string, error) {
	vms, err := s.inv.ListHostVMs(hostSystemID)
	if err != nil {
		return false, nil, err
	}
	// Gate A: host empty
	if len(vms) == 0 {
		return true, nil, nil
	}

	// Compute affected clusters from VMs on host
	affectedCounts := map[string]int{}
	for _, vm := range vms {
		if !maint.IsControlPlaneVM(vm.Tags) {
			continue
		}
		if cid, ok := maint.ClusterIDFromTags(vm.Tags); ok {
			affectedCounts[cid] = affectedCounts[cid] + 1
		}
	}
	var affected []string
	for k := range affectedCounts {
		affected = append(affected, k)
	}

	// Gate B: For each affected cluster, ensure ready replacements count >= affected count, on different hosts
	for _, cid := range affected {
		params := maasclient.ParamsBuilder().
			Set(maasclient.TagKey, maint.TagVMControlPlane).
			Set(maasclient.TagKey, maint.TagVMClusterPrefix+maint.SanitizeID(cid)).
			Set(maasclient.TagKey, maint.BuildReadyOpTag(opID))
		vmlist, err := s.maas.Machines().List(ctx, params)
		if err != nil {
			return false, affected, err
		}
		readyDifferentHost := 0
		seen := map[string]struct{}{}
		for _, m := range vmlist {
			dm, derr := m.Get(ctx)
			if derr != nil {
				continue
			}
			parent := dm.Parent()
			if parent == "" || parent == hostSystemID {
				continue
			}
			// count unique replacements
			if _, ok := seen[dm.SystemID()]; !ok {
				seen[dm.SystemID()] = struct{}{}
				readyDifferentHost++
			}
		}
		if readyDifferentHost < affectedCounts[cid] {
			log.V(1).Info("Insufficient ready replacements on different hosts", "cluster", cid, "needed", affectedCounts[cid], "readyDifferentHost", readyDifferentHost, "host", hostSystemID)
			return false, affected, nil
		}
	}

	return true, affected, nil
}

// ClearMaintenanceTags removes op/maintenance/noschedule tags from host.
func (s *HostMaintenanceService) ClearMaintenanceTags(ctx context.Context, hostSystemID, opID string) error {
	if err := maint.ClearHostMaintenanceTags(s.tags, hostSystemID, opID, true); err != nil {
		return err
	}
	return nil
}
