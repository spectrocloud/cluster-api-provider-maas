package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/util"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/util/trust"

	// embed template
	_ "embed"
)

//go:embed templates/lxd_initializer_ds.yaml
var lxdInitTemplate string

//go:embed templates/lxd_initializer_rbac.yaml
var lxdInitRBACTemplate string

func render(replacements map[string]string, tmpl string) string {
	for k, v := range replacements {
		tmpl = strings.ReplaceAll(tmpl, k, v)
	}
	return tmpl
}

// ensureLXDInitializerDS creates or deletes the DaemonSet that initialises LXD on all nodes (control-plane and worker)
func (r *MaasClusterReconciler) ensureLXDInitializerDS(ctx context.Context, clusterScope *scope.ClusterScope) error {
	cluster := clusterScope.MaasCluster

	dsName := fmt.Sprintf("lxd-initializer-%s", cluster.Name)
	dsNamespace := cluster.Namespace

	// Always operate on the TARGET cluster client
	remoteClient, err := r.getTargetClient(ctx, clusterScope)
	if err != nil {
		return err
	}

	// If feature is off or cluster is being deleted, we're done
	if !clusterScope.IsLXDHostEnabled() || !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	// Gate: ensure pivot completed. Require mgmt namespace to have clusterEnv=target
	isTarget, clusterEnv := r.namespaceIsTarget(ctx, dsNamespace)
	if !isTarget {
		r.Log.Info("Namespace not marked as target; deferring LXD initializer", "namespace", dsNamespace, "clusterEnv", clusterEnv)
		return nil
	}

	// Gate: derive desired CP count from MaasCloudConfig; fallback to KCP
	desiredCP, _ := r.computeDesiredControlPlane(ctx, dsNamespace, cluster.Name)

	// New gate: proceed if any node needs initialization. This avoids deadlock during upgrades
	// when an old node is NotReady due to HMC constraints but new nodes must be initialized.
	if !r.anyNodeNeedsInitialization(ctx, remoteClient) {
		r.Log.Info("All nodes already labeled initialized; considering DS cleanup", "namespace", dsNamespace, "ds", dsName)
		if done, err := r.maybeShortCircuitDelete(ctx, remoteClient, dsNamespace, desiredCP, dsName); err != nil {
			r.Log.Error(err, "failed to maybe short circuit delete", "namespace", dsNamespace, "ds", dsName)
			return err
		} else if done {
			r.Log.Info("deleted existing initializer DS - all nodes are ready and initialized", "namespace", dsNamespace, "ds", dsName)
			return nil
		}
		r.Log.Info("no nodes need initialization; skipping DS creation", "namespace", dsNamespace, "ds", dsName)
		return nil
	}

	//TODO: Check if we require this
	// if ok := r.enoughNodesReady(ctx, remoteClient, desiredCP, readyByKCP); !ok {
	// 	return nil
	// }

	if err := r.deleteExistingInitializerDS(ctx, remoteClient, dsNamespace); err != nil {
		r.Log.Error(err, "failed to delete existing initializer DS", "namespace", dsNamespace, "ds", dsName)
		return err
	}

	// Ensure RBAC resources are created on the target cluster
	if err := r.ensureLXDInitializerRBACOnTarget(ctx, remoteClient, dsNamespace); err != nil {
		r.Log.Error(err, "failed to ensure LXD initializer RBAC", "namespace", dsNamespace, "ds", dsName)
		return fmt.Errorf("failed to ensure LXD initializer RBAC: %v", err)
	}

	if done, err := r.maybeShortCircuitDelete(ctx, remoteClient, dsNamespace, desiredCP, dsName); err != nil {
		r.Log.Error(err, "failed to maybe short circuit delete", "namespace", dsNamespace, "ds", dsName)
		return err
	} else if done {
		r.Log.Info("deleted existing initializer DS - all nodes are ready and initialized", "namespace", dsNamespace, "ds", dsName)
		return nil
	}

	ds, err := r.renderDaemonSetForCluster(clusterScope, dsName, dsNamespace)
	if err != nil {
		r.Log.Error(err, "failed to render DaemonSet for cluster", "namespace", dsNamespace, "ds", dsName)
		return err
	}

	// Do not set owner refs across clusters; just create/patch on target cluster.
	// Mutate existing DaemonSet so changes to template/spec take effect on reconcile.
	current := &appsv1.DaemonSet{}
	current.Name = dsName
	current.Namespace = dsNamespace

	_, err = controllerutil.CreateOrPatch(ctx, remoteClient, current, func() error {
		// Preserve immutable selector if already present; align labels.
		current.Labels = ds.Labels
		current.Annotations = ds.Annotations

		// Update pod template and mutable spec fields
		current.Spec.Template = ds.Spec.Template
		current.Spec.UpdateStrategy = ds.Spec.UpdateStrategy
		current.Spec.MinReadySeconds = ds.Spec.MinReadySeconds
		current.Spec.RevisionHistoryLimit = ds.Spec.RevisionHistoryLimit

		// Initialize selector if missing (only valid on create)
		if current.Spec.Selector == nil || len(current.Spec.Selector.MatchLabels) == 0 {
			current.Spec.Selector = ds.Spec.Selector
		}
		// Ensure template labels include selector labels
		if current.Spec.Selector != nil && len(current.Spec.Selector.MatchLabels) > 0 {
			if current.Spec.Template.Labels == nil {
				current.Spec.Template.Labels = map[string]string{}
			}
			for k, v := range current.Spec.Selector.MatchLabels {
				current.Spec.Template.Labels[k] = v
			}
		}
		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to create/patch DaemonSet", "namespace", dsNamespace, "ds", dsName)
		return err
	}
	r.Log.Info("created/patched DaemonSet", "namespace", dsNamespace, "ds", dsName)
	return nil
}

// ensureLXDInitializerRBACOnTarget creates the RBAC resources for lxd-initializer on the target cluster
func (r *MaasClusterReconciler) ensureLXDInitializerRBACOnTarget(ctx context.Context, remoteClient client.Client, namespace string) error {
	// Parse RBAC template into separate resources
	rbacYaml := strings.ReplaceAll(lxdInitRBACTemplate, "namespace: default", fmt.Sprintf("namespace: %s", namespace))

	// Split the YAML into separate documents
	docs := strings.Split(rbacYaml, "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Parse as unstructured object to handle different resource types
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), obj); err != nil {
			return fmt.Errorf("failed to unmarshal RBAC resource: %v", err)
		}

		// Set namespace for namespaced resources
		if obj.GetKind() == "ServiceAccount" {
			obj.SetNamespace(namespace)
		}

		// Create or update the resource on target cluster
		_, err := controllerutil.CreateOrPatch(ctx, remoteClient, obj, func() error { return nil })
		if err != nil {
			return fmt.Errorf("failed to create/patch %s %s: %v", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

// getTargetClient returns the workload cluster client or a wrapped error
func (r *MaasClusterReconciler) getTargetClient(ctx context.Context, clusterScope *scope.ClusterScope) (client.Client, error) {
	remoteClient, err := clusterScope.GetWorkloadClusterClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get target cluster client: %v", err)
	}
	return remoteClient, nil
}

// namespaceIsTarget checks if the management namespace is annotated as clusterEnv=target
func (r *MaasClusterReconciler) namespaceIsTarget(ctx context.Context, namespace string) (bool, string) {
	mgmtNS := &corev1.Namespace{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: namespace}, mgmtNS); err != nil {
		return false, ""
	}
	if mgmtNS.Annotations == nil {
		return false, ""
	}
	v := strings.TrimSpace(mgmtNS.Annotations["clusterEnv"])
	return v == "target", v
}

// computeDesiredControlPlane determines desired control-plane replicas and ready count from KubeadmControlPlane
func (r *MaasClusterReconciler) computeDesiredControlPlane(ctx context.Context, namespace, clusterName string) (int32, int32) {
	desiredCP := int32(1)
	readyByKCP := int32(0)

	// Use KubeadmControlPlane as the authoritative source
	kcpList := &unstructured.UnstructuredList{}
	kcpList.SetGroupVersionKind(schema.GroupVersionKind{Group: "controlplane.cluster.x-k8s.io", Version: "v1beta1", Kind: "KubeadmControlPlaneList"})
	if err := r.Client.List(ctx, kcpList, client.InNamespace(namespace), client.MatchingLabels{
		"cluster.x-k8s.io/cluster-name": clusterName,
	}); err == nil {
		if len(kcpList.Items) > 0 {
			item := kcpList.Items[0]
			if v, found, _ := unstructured.NestedInt64(item.Object, "spec", "replicas"); found && v > 0 {
				desiredCP = util.SafeInt64ToInt32(v)
			}
			if v, found, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas"); found && v >= 0 {
				readyByKCP = util.SafeInt64ToInt32(v)
			}
		}
	}

	return desiredCP, readyByKCP
}

// enoughNodesReady checks the target cluster for Ready nodes (both control-plane and worker)
func (r *MaasClusterReconciler) enoughNodesReady(ctx context.Context, remoteClient client.Client, desiredCP, readyByKCP int32) bool {
	nodeList := &corev1.NodeList{}
	// Check all nodes, not just control-plane
	if err := remoteClient.List(ctx, nodeList); err == nil {
		ready := 0
		for _, n := range nodeList.Items {
			for _, c := range n.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					ready++
					break
				}
			}
		}
		// Require at least the desired CP count of nodes to be ready
		if int64(len(nodeList.Items)) < int64(desiredCP) || int64(ready) < int64(desiredCP) {
			r.Log.Info("Not enough nodes present/ready yet; skipping DS for now", "desiredCP", desiredCP, "readyByKCP", readyByKCP, "nodeList", len(nodeList.Items), "ready", ready)
			return false
		}
	}
	return true
}

// anyNodeNeedsInitialization returns true if any node (control-plane or worker) needs initialization.
// Only checks Ready nodes to avoid false positives from nodes that aren't ready yet.
func (r *MaasClusterReconciler) anyNodeNeedsInitialization(ctx context.Context, remoteClient client.Client) bool {
	nodeList := &corev1.NodeList{}
	// Check all nodes, not just control-plane, to include worker nodes
	if err := remoteClient.List(ctx, nodeList); err != nil {
		r.Log.Info("Failed to list nodes; proceeding to create initializer DS to be safe", "error", err)
		return true
	}
	if len(nodeList.Items) == 0 {
		r.Log.Info("No nodes reported yet; proceeding with initializer DS")
		return true
	}

	for _, n := range nodeList.Items {
		// Check if node is Ready
		isReady := false
		for _, condition := range n.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				isReady = true
				break
			}
		}

		// If node is Ready but not initialized, it needs initialization
		if isReady && (n.Labels == nil || n.Labels["lxdhost.cluster.com/initialized"] != "true") {
			// Determine node role for logging
			nodeRole := "worker"
			if n.Labels != nil && n.Labels["node-role.kubernetes.io/control-plane"] != "" {
				nodeRole = "control-plane"
			}
			r.Log.Info("Ready node requires LXD initialization", "node", n.Name, "role", nodeRole)
			return true
		}

		// If node is not Ready yet but has the label, log a warning (might be stale)
		if !isReady && n.Labels != nil && n.Labels["lxdhost.cluster.com/initialized"] == "true" {
			r.Log.Info("Node has initialization label but is not Ready - may need re-initialization", "node", n.Name)
		}
	}
	return false
}

// deleteExistingInitializerDS removes any DaemonSets with old labeling in the namespace
func (r *MaasClusterReconciler) deleteExistingInitializerDS(ctx context.Context, remoteClient client.Client, namespace string) error {
	dsList := &appsv1.DaemonSetList{}
	if err := remoteClient.List(ctx, dsList, client.InNamespace(namespace), client.MatchingLabels{
		"app": "lxd-initializer",
	}); err != nil {
		return fmt.Errorf("failed to list DaemonSets: %v", err)
	}

	for _, ds := range dsList.Items {
		if err := remoteClient.Delete(ctx, &ds); err != nil {
			return fmt.Errorf("failed to delete DaemonSet %s: %v", ds.Name, err)
		}
	}
	return nil
}

// maybeShortCircuitDelete deletes the DS if all nodes are already initialized
// BUT only if we have exactly desiredCP nodes - avoids deleting during maintenance when new nodes are joining
func (r *MaasClusterReconciler) maybeShortCircuitDelete(ctx context.Context, remoteClient client.Client, namespace string, desiredCP int32, dsName string) (bool, error) {
	shortCircuitNodes := &corev1.NodeList{}
	// Check all nodes, not just control-plane
	if err := remoteClient.List(ctx, shortCircuitNodes); err != nil || len(shortCircuitNodes.Items) == 0 {
		return false, nil
	}

	initCount := 0
	readyCount := 0
	for _, n := range shortCircuitNodes.Items {
		// Check if node is Ready
		for _, condition := range n.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				readyCount++
				break
			}
		}

		// Check if node is initialized
		if n.Labels != nil && n.Labels["lxdhost.cluster.com/initialized"] == "true" {
			initCount++
		}
	}

	// Delete initializer DS only when ALL nodes (control-plane + worker) are initialized.
	// This matches the new requirement to register both CP and worker nodes.
	totalNodes := len(shortCircuitNodes.Items)
	if totalNodes > 0 && initCount == totalNodes {
		shortCircuitDSList := &appsv1.DaemonSetList{}
		if err := remoteClient.List(ctx, shortCircuitDSList, client.InNamespace(namespace), client.MatchingLabels{"app": dsName}); err == nil {
			for _, ds := range shortCircuitDSList.Items {
				_ = remoteClient.Delete(ctx, &ds)
			}
		}
		r.Log.Info("Deleted LXD initializer DaemonSet - all nodes initialized",
			"desiredCP", desiredCP, "totalNodes", totalNodes, "readyNodes", readyCount, "initializedNodes", initCount)
		return true, nil
	}
	return false, nil
}

// renderDaemonSetForCluster renders the DS YAML from template using cluster config and returns a DaemonSet object
func (r *MaasClusterReconciler) renderDaemonSetForCluster(clusterScope *scope.ClusterScope, dsName, namespace string) (*appsv1.DaemonSet, error) {
	cluster := clusterScope.MaasCluster
	cfg := clusterScope.GetLXDConfig()

	sb := cfg.StorageBackend
	if sb == "" {
		sb = "zfs"
	}
	ss := cfg.StorageSize
	if ss == "" {
		ss = "50"
	}
	nb := cfg.NetworkBridge
	skip := "true"
	if cfg.SkipNetworkUpdate != nil && !*cfg.SkipNetworkUpdate {
		skip = "false"
	}

	nt := cfg.NICType
	if nt == "" {
		nt = "bridged"
	}
	np := cfg.NICParent
	// Deterministic per-cluster trust password derived from cluster UID
	tp := trust.DeriveTrustPassword(string(cluster.UID))

	rendered := render(map[string]string{
		"${STORAGE_BACKEND}":     sb,
		"${STORAGE_SIZE}":        ss,
		"${NETWORK_BRIDGE}":      nb,
		"${SKIP_NETWORK_UPDATE}": skip,
		"${TRUST_PASSWORD}":      tp,
		"${NIC_TYPE}":            nt,
		"${NIC_PARENT}":          np,
	}, lxdInitTemplate)

	dsYaml := strings.ReplaceAll(rendered, "${DS_NAME}", dsName)

	ds := &appsv1.DaemonSet{}
	if err := yaml.Unmarshal([]byte(dsYaml), ds); err != nil {
		return nil, err
	}

	// ensure names/labels are cluster-specific without touching the image name
	ds.Name = dsName
	if ds.Labels == nil {
		ds.Labels = map[string]string{}
	}
	ds.Labels["app"] = dsName
	ds.Spec.Selector.MatchLabels["app"] = dsName
	ds.Spec.Template.Labels["app"] = dsName
	ds.Namespace = namespace

	return ds, nil
}
