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
package maintenance

// EnsureHostMaintenanceTags ensures the standard HMC tags exist and are
// assigned to the specified host. This operation is idempotent.
//
// Tags ensured/assigned:
// - TagHostMaintenance (drain source)
// - TagHostNoSchedule (prevent new placements)
// - BuildOpTag(opID) (session marker)
func EnsureHostMaintenanceTags(tags TagService, hostSystemID, opID string) error {
	// Ensure inventory tags exist (idempotent create)
	if err := tags.EnsureTagInInventory(TagHostMaintenance); err != nil {
		return err
	}
	if err := tags.EnsureTagInInventory(TagHostNoSchedule); err != nil {
		return err
	}
	opTag := BuildOpTag(opID)
	if err := tags.EnsureTagInInventory(opTag); err != nil {
		return err
	}

	// Assign to host (idempotent assign)
	if err := tags.AddTagToHost(hostSystemID, TagHostMaintenance); err != nil {
		return err
	}
	if err := tags.AddTagToHost(hostSystemID, TagHostNoSchedule); err != nil {
		return err
	}
	if err := tags.AddTagToHost(hostSystemID, opTag); err != nil {
		return err
	}
	return nil
}

// EnsureHostReadyAckTag ensures the per-session readiness acknowledgement tag
// exists and is assigned to the host. This allows HMC to coordinate handoff
// with VEC across clusters.
func EnsureHostReadyAckTag(tags TagService, hostSystemID, clusterID, opID string) error {
	readyTag := BuildReadyHostTag(clusterID, opID)
	if err := tags.EnsureTagInInventory(readyTag); err != nil {
		return err
	}
	if err := tags.AddTagToHost(hostSystemID, readyTag); err != nil {
		return err
	}
	return nil
}

// ClearHostMaintenanceTags removes the session op tag from the host. Depending
// on gating policy, callers may remove the maintenance and noschedule tags here
// or in a dedicated finalization phase (PCP-5339).
func ClearHostMaintenanceTags(tags TagService, hostSystemID, opID string, removeMaintenance bool) error {
	opTag := BuildOpTag(opID)
	if err := tags.RemoveTagFromHost(hostSystemID, opTag); err != nil {
		return err
	}
	if removeMaintenance {
		if err := tags.RemoveTagFromHost(hostSystemID, TagHostNoSchedule); err != nil {
			return err
		}
		if err := tags.RemoveTagFromHost(hostSystemID, TagHostMaintenance); err != nil {
			return err
		}
	}
	return nil
}
