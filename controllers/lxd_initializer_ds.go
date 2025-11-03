package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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

// ensureLXDInitializerDS creates or deletes the DaemonSet that initialises LXD on control-plane nodes
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

	// Gate: derive desired CP count from KubeadmControlPlane
	desiredCP, readyByKCP := r.computeDesiredControlPlane(ctx, dsNamespace, cluster.Name)

	if ok := r.enoughCPNodesReady(ctx, remoteClient, desiredCP, readyByKCP); !ok {
		return nil
	}

	if err := r.deleteExistingInitializerDS(ctx, remoteClient, dsNamespace); err != nil {
		return err
	}

	// Ensure RBAC resources are created on target cluster
	if err := r.ensureLXDInitializerRBACOnTarget(ctx, remoteClient, dsNamespace); err != nil {
		return fmt.Errorf("failed to ensure LXD initializer RBAC: %v", err)
	}

	if done, err := r.maybeShortCircuitDelete(ctx, remoteClient, dsNamespace, desiredCP, dsName); err != nil {
		return err
	} else if done {
		return nil
	}

	ds, err := r.renderDaemonSetForCluster(clusterScope, dsName, dsNamespace)
	if err != nil {
		return err
	}

	// Do not set owner refs across clusters; just create/patch on target cluster
	_, err = controllerutil.CreateOrPatch(ctx, remoteClient, ds, func() error { return nil })
	return err
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

// enoughCPNodesReady checks the target cluster for Ready control-plane nodes
func (r *MaasClusterReconciler) enoughCPNodesReady(ctx context.Context, remoteClient client.Client, desiredCP, readyByKCP int32) bool {
	nodeList := &corev1.NodeList{}
	cpSelector := labels.SelectorFromSet(labels.Set{
		"node-role.kubernetes.io/control-plane": "",
	})
	if err := remoteClient.List(ctx, nodeList, &client.ListOptions{LabelSelector: cpSelector}); err == nil {
		ready := 0
		for _, n := range nodeList.Items {
			for _, c := range n.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					ready++
					break
				}
			}
		}
		if int64(len(nodeList.Items)) < int64(desiredCP) || int64(ready) < int64(desiredCP) {
			r.Log.Info("Not enough control-plane nodes present/ready yet; skipping DS for now", "desiredCP", desiredCP, "readyByKCP", readyByKCP, "nodeList", len(nodeList.Items), "ready", ready)
			return false
		}
	}
	return true
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

// maybeShortCircuitDelete deletes the DS if all CP nodes are already initialized
func (r *MaasClusterReconciler) maybeShortCircuitDelete(ctx context.Context, remoteClient client.Client, namespace string, desiredCP int32, dsName string) (bool, error) {
	shortCircuitNodes := &corev1.NodeList{}
	shortCircuitSelector := labels.SelectorFromSet(labels.Set{
		"node-role.kubernetes.io/control-plane": "",
	})
	if err := remoteClient.List(ctx, shortCircuitNodes, &client.ListOptions{LabelSelector: shortCircuitSelector}); err != nil || len(shortCircuitNodes.Items) == 0 {
		return false, nil
	}

	initCount := 0
	for _, n := range shortCircuitNodes.Items {
		if n.Labels != nil && n.Labels["lxdhost.cluster.com/initialized"] == "true" {
			initCount++
		}
	}
	if int64(len(shortCircuitNodes.Items)) >= int64(desiredCP) && int64(initCount) >= int64(desiredCP) {
		shortCircuitDSList := &appsv1.DaemonSetList{}
		if err := remoteClient.List(ctx, shortCircuitDSList, client.InNamespace(namespace), client.MatchingLabels{"app": dsName}); err == nil {
			for _, ds := range shortCircuitDSList.Items {
				_ = remoteClient.Delete(ctx, &ds)
			}
		}
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
