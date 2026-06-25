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

// This spec reuses the shared CAPI QuickStartSpec, but provisions the workload cluster from the
// "lxd" flavor: the control-plane machine is created as an LXD VM scheduled onto a pre-existing
// LXD host in MAAS (spec.lxd.enabled=true), while the workers remain bare-metal. It verifies the
// WLC path comes up healthy (control plane initialized, CNI applied via ClusterResourceSet, all
// machines and nodes Ready).
//
// Prerequisite: the target MAAS must already have registered LXD hosts (from an HCP / host
// control-plane cluster) with allocatable capacity in the configured FAILURE_DOMAIN. The e2e
// suite does not provision those hosts.
var _ = Describe("When following the Cluster API quick-start on LXD VMs [WLC]", Label("WLC"), func() {
	capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
		return capi_e2e.QuickStartSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
			Flavor:                ptr.To("lxd"),
		}
	})
})
