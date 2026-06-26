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
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/lxd"
)

// This spec reuses the shared CAPI QuickStartSpec to provision an HCP (Host Control Plane) cluster
// from the "hcp" flavor: the MaasCluster sets spec.lxdConfig.enabled=true, so its bare-metal
// control-plane and worker machines are registered as LXD VM hosts.
//
// HCP clusters MUST use Cilium, not Calico (Calico tears down the bridge networking on the PXE-boot
// interface the lxd-initializer relies on). Unlike the default/lxd flavors — which deliver Calico
// through a ClusterResourceSet baked into the cluster-template — Cilium is applied directly to the
// workload cluster in the WaitForControlPlaneInitialized hook (the same mechanism the framework's
// built-in CNIManifestPath uses). This is required because Cilium's manifest contains runtime shell
// tokens (e.g. "${BIN_PATH}") that clusterctl's template envsubst would reject if the manifest were
// routed through a cluster-template CRS.
//
// Beyond the standard "cluster is healthy" checks, PostMachinesProvisioned verifies the LXD
// host-registration flow completed end-to-end: MaasCluster reports LXDReady=True and every workload
// node carries the lxd-initializer's "initialized" label. The lxd-initializer DaemonSet is deployed
// automatically by the controller, so the test does not apply it.
//
// Prerequisite: the target MAAS must have bare-metal machines eligible for LXD hosting in
// FAILURE_DOMAIN (sufficient CPU/memory/disk for the zfs storage pool).
var _ = Describe("When following the Cluster API quick-start for an LXD host cluster [HCP]", Label("HCP"), func() {
	capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
		return capi_e2e.QuickStartSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
			Flavor:                ptr.To("hcp"),
			ControlPlaneWaiters: clusterctl.ControlPlaneWaiters{
				// Replicate the default control-plane-initialized wait, then install Cilium on the
				// workload cluster before QuickStartSpec waits for nodes to become Ready (nodes only
				// go Ready once a CNI is present).
				WaitForControlPlaneInitialized: func(ctx context.Context, input clusterctl.ApplyCustomClusterTemplateAndWaitInput, result *clusterctl.ApplyCustomClusterTemplateAndWaitResult) {
					result.ControlPlane = framework.DiscoveryAndWaitForControlPlaneInitialized(ctx, framework.DiscoveryAndWaitForControlPlaneInitializedInput{
						Lister:  input.ClusterProxy.GetClient(),
						Cluster: result.Cluster,
					}, input.WaitForControlPlaneIntervals...)

					installCilium(input.ClusterProxy.GetWorkloadCluster(ctx, result.Cluster.Namespace, result.Cluster.Name))
				},
			},
			PostMachinesProvisioned: func(mgmt framework.ClusterProxy, ns, clusterName string) {
				assertLXDHostsReady(mgmt, ns, clusterName)
			},
		}
	})
})

// installCilium applies the Cilium manifest (referenced by the CILIUM config variable, rendered with
// `helm template` — see data/cni/cilium/cilium.yaml) directly to the workload cluster. Applying the
// raw YAML via the workload ClusterProxy (rather than baking it into the cluster-template) keeps the
// manifest out of clusterctl's envsubst, so Cilium's runtime "${BIN_PATH}" tokens are preserved.
func installCilium(workload framework.ClusterProxy) {
	ciliumPath := e2eConfig.MustGetVariable(ciliumResourcesVar)
	Byf("Installing Cilium from %q onto the workload cluster", ciliumPath)
	cniYAML, err := os.ReadFile(ciliumPath) //nolint:gosec
	Expect(err).ToNot(HaveOccurred(), "Failed to read the Cilium manifest %q", ciliumPath)
	Expect(workload.CreateOrUpdate(ctx, cniYAML)).To(Succeed(), "Failed to apply Cilium to the workload cluster")
}

// assertLXDHostsReady verifies the HCP LXD host-registration flow completed: the MaasCluster reports
// LXDReady=True, and every node in the workload cluster carries the lxd-initializer's label. These
// lag node-Ready (the controller deploys the lxd-initializer DaemonSet only after the control plane
// is up, then each node installs LXD and registers with MAAS), so we poll with the wait-lxd-ready
// interval.
func assertLXDHostsReady(mgmt framework.ClusterProxy, ns, clusterName string) {
	intervals := e2eConfig.GetIntervals("hcp", "wait-lxd-ready")

	Byf("Waiting for MaasCluster %s/%s to report LXDReady=True", ns, clusterName)
	Eventually(func(g Gomega) {
		maasCluster := &infrav1.MaasCluster{}
		g.Expect(mgmt.GetClient().Get(ctx, client.ObjectKey{Namespace: ns, Name: clusterName}, maasCluster)).To(Succeed())
		cond := meta.FindStatusCondition(maasCluster.Status.Conditions, infrav1.LXDReadyCondition)
		g.Expect(cond).ToNot(BeNil(), "MaasCluster is missing the %s condition", infrav1.LXDReadyCondition)
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue), "%s condition is not True (reason=%s, message=%s)", infrav1.LXDReadyCondition, cond.Reason, cond.Message)
	}, intervals...).Should(Succeed())

	Byf("Waiting for every node in workload cluster %s/%s to be LXD-host initialized", ns, clusterName)
	workloadClient := mgmt.GetWorkloadCluster(ctx, ns, clusterName).GetClient()
	Eventually(func(g Gomega) {
		nodes := &corev1.NodeList{}
		g.Expect(workloadClient.List(ctx, nodes)).To(Succeed())
		g.Expect(nodes.Items).ToNot(BeEmpty(), "workload cluster has no nodes")
		for i := range nodes.Items {
			g.Expect(nodes.Items[i].Labels).To(HaveKeyWithValue(lxd.LXDHostInitializedLabel, "true"),
				"node %s is not marked LXD-host initialized", nodes.Items[i].Name)
		}
	}, intervals...).Should(Succeed())
}
