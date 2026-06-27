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
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

// This spec upgrades a WLC (LXD workload cluster) — control plane as an LXD VM — from
// KUBERNETES_VERSION_UPGRADE_FROM to KUBERNETES_VERSION_UPGRADE_TO via the "lxd-upgrades" flavor,
// which ships target-version MaasMachineTemplates (with the LXD VM block on the control plane).
var _ = Describe("When upgrading a WLC's Kubernetes version [Upgrade] [WLC]", Label("Upgrade", "WLC"), func() {
	capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
		return capi_e2e.ClusterUpgradeConformanceSpecInput{
			E2EConfig:                e2eConfig,
			ClusterctlConfigPath:     clusterctlConfigPath,
			BootstrapClusterProxy:    bootstrapClusterProxy,
			ArtifactFolder:           artifactFolder,
			SkipCleanup:              skipCleanup,
			SkipConformanceTests:     true,
			Flavor:                   ptr.To("lxd-upgrades"),
			ControlPlaneMachineCount: machineCountFromConfig("CONTROL_PLANE_MACHINE_COUNT", 1),
			WorkerMachineCount:       machineCountFromConfig("WORKER_MACHINE_COUNT", 1),
		}
	})
})
