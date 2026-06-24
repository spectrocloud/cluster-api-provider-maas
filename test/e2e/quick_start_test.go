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
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

// This spec reuses the shared CAPI QuickStartSpec to provision a MAAS workload cluster from the
// default cluster-template flavor and verify it comes up healthy (control plane initialized, CNI
// applied via ClusterResourceSet, all machines and nodes Ready).
var _ = Describe("When following the Cluster API quick-start [PR-Blocking] [QuickStart]", func() {
	capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
		return capi_e2e.QuickStartSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
		}
	})
})
