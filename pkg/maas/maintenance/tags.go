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

import "strings"

const (
	// TagHostMaintenance marks a host as under maintenance (drain source).
	TagHostMaintenance = "maas.lxd-host-maintenance"
	// TagHostNoSchedule prevents new placements on the host during maintenance.
	TagHostNoSchedule = "maas.lxd-host-noschedule"
	// TagHostOpPrefix prefixes the maintenance session opId stored as a tag.
	TagHostOpPrefix = "maas.lxd-hcp-op-"

	// TagVMControlPlane marks a VM as a control-plane VM.
	TagVMControlPlane = "maas.lxd-wlc-cp"
	// TagVMClusterPrefix prefixes the owning WLC cluster identifier.
	TagVMClusterPrefix = "maas.lxd-wlc-"
	// TagVMReadyOpPrefix prefixes the ready acknowledgement for a given session opId.
	TagVMReadyOpPrefix = "maas.lxd-ready-op-"
)

// BuildOpTag builds the session opId tag for hosts.
func BuildOpTag(opID string) string { return TagHostOpPrefix + opID }

// ParseOpTag returns the opId from a slice of tags if present.
func ParseOpTag(in []string) (string, bool) {
	for _, t := range in {
		if strings.HasPrefix(t, TagHostOpPrefix) && len(t) > len(TagHostOpPrefix) {
			return t[len(TagHostOpPrefix):], true
		}
	}
	return "", false
}
