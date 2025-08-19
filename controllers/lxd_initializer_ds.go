package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maas/scope"

	// embed template
	_ "embed"
)

//go:embed templates/lxd_initializer_ds.yaml
var lxdInitTemplate string

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

	// First clean up any existing DaemonSets in this namespace
	dsList := &appsv1.DaemonSetList{}
	if err := r.Client.List(ctx, dsList, client.InNamespace(dsNamespace), client.MatchingLabels{
		"app": "lxd-initializer",
	}); err != nil {
		return fmt.Errorf("failed to list DaemonSets: %v", err)
	}

	// Delete all existing LXD initializer DaemonSets
	for _, ds := range dsList.Items {
		if err := r.Client.Delete(ctx, &ds); err != nil {
			return fmt.Errorf("failed to delete DaemonSet %s: %v", ds.Name, err)
		}
	}

	// If feature is off or cluster is being deleted, we're done after cleanup
	if !clusterScope.IsLXDHostEnabled() || !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	// pull LXD config
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
	//if cfg.SkipNetworkUpdate != nil && !*cfg.SkipNetworkUpdate {
	//	skip = "false"
	//}

	nt := cfg.NICType
	np := cfg.NICParent
	tp := "capmaas"

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
		return err
	}

	// ensure names/labels are cluster-specific without touching the image name
	ds.Name = dsName
	if ds.Labels == nil {
		ds.Labels = map[string]string{}
	}
	ds.Labels["app"] = dsName
	ds.Spec.Selector.MatchLabels["app"] = dsName
	ds.Spec.Template.Labels["app"] = dsName
	ds.Namespace = dsNamespace

	if err := controllerutil.SetControllerReference(cluster, ds, r.Scheme); err != nil {
		return err
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, ds, func() error { return nil })
	return err
}
