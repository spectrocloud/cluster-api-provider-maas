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

const (
	// Host-level tags written by HMC
	HostMaintenance = "maas.lxd-host-maintenance"
	HostNoSchedule  = "maas.lxd-host-noschedule"
	HostOpPrefix    = "maas.lxd-hcp-op-"

	// VM-level tags written by VEC on replacement CP VM after health
	VMCP            = "maas.lxd-wlc-cp"
	VMClusterPrefix = "maas.lxd-wlc-"
	VMReadyOpPrefix = "maas.lxd-ready-op-"
)

// BuildOpTag builds the session opId tag for hosts.
func BuildOpTag(uuid string) string { return HostOpPrefix + uuid }

// ParseOpTag returns the opId from a slice of tags if present.
func ParseOpTag(in []string) (string, bool) {
	for _, t := range in {
		if len(t) > len(HostOpPrefix) && t[:len(HostOpPrefix)] == HostOpPrefix {
			return t[len(HostOpPrefix):], true
		}
	}
	return "", false
}
