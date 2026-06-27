//go:build e2e
// +build e2e

/*
Copyright 2024 SpectroCloud.

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

package e2e

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

// machineCountFromConfig reads an integer machine-count variable (e.g. WORKER_MACHINE_COUNT) from the
// e2e config / OS env and returns it as a *int64. ClusterUpgradeConformanceSpec defaults the worker
// count to 2 when its input leaves WorkerMachineCount nil and passes it explicitly to clusterctl,
// which would override the exported WORKER_MACHINE_COUNT; passing the count through keeps the
// upgrade specs honoring the same counts as the quick-start specs.
func machineCountFromConfig(varName string, fallback int64) *int64 {
	if !e2eConfig.HasVariable(varName) {
		return ptr.To(fallback)
	}
	raw := e2eConfig.MustGetVariable(varName)
	n, err := strconv.ParseInt(raw, 10, 64)
	Expect(err).ToNot(HaveOccurred(), "%s=%q is not a valid integer", varName, raw)
	return ptr.To(n)
}

// This spec reuses the shared CAPI ClusterUpgradeConformanceSpec to upgrade a standard (bare-metal)
// MAAS workload cluster from KUBERNETES_VERSION_UPGRADE_FROM to KUBERNETES_VERSION_UPGRADE_TO. Because
// a MaasMachineTemplate's image is immutable and encodes the K8s version, the "upgrades" flavor ships
// a second pair of MaasMachineTemplates (control-plane-upgrade / md-0-upgrade) carrying the target
// image; the spec repoints the KCP/MachineDeployment to them (via CONTROL_PLANE_MACHINE_TEMPLATE_UPGRADE_TO
// / WORKERS_MACHINE_TEMPLATE_UPGRADE_TO) and bumps the version. Conformance (kubetest) is skipped.
var _ = Describe("When upgrading a workload cluster's Kubernetes version [Upgrade]", Label("Upgrade"), func() {
	capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
		return capi_e2e.ClusterUpgradeConformanceSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
			SkipConformanceTests:  true,
			Flavor:                ptr.To("upgrades"),
			// Honor CONTROL_PLANE_MACHINE_COUNT / WORKER_MACHINE_COUNT (otherwise the spec forces 2 workers).
			ControlPlaneMachineCount: machineCountFromConfig("CONTROL_PLANE_MACHINE_COUNT", 1),
			WorkerMachineCount:       machineCountFromConfig("WORKER_MACHINE_COUNT", 1),
		}
	})
})
