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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var maasClusterGVK = schema.GroupVersionKind{
	Group:   "infrastructure.cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "MaasCluster",
}

// MigrateMaasClusterFailureDomains rewrites any MaasCluster whose
// status.failureDomains is still stored as a map (the pre-v0.9.0 shape) into an
// empty slice, so the typed controller can List them under the v0.9.0 schema
// where status.failureDomains is []FailureDomain. The field is derived state,
// rebuilt from spec.failureDomains on the next reconcile
// (see maascluster_controller.go), so clearing it loses nothing.
//
// It runs once at startup, before the manager's typed cache starts: an
// unstructured List tolerates the stored map shape that crashes a typed List
// ("cannot unmarshal object into Go struct field ... of type []FailureDomain").
// Idempotent — objects already holding a slice (or with no failureDomains) are skipped.
func MigrateMaasClusterFailureDomains(ctx context.Context, c client.Client, log logr.Logger) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   maasClusterGVK.Group,
		Version: maasClusterGVK.Version,
		Kind:    maasClusterGVK.Kind + "List",
	})
	if err := c.List(ctx, list); err != nil {
		return err
	}

	patch := client.RawPatch(types.MergePatchType, []byte(`{"status":{"failureDomains":[]}}`))
	for i := range list.Items {
		item := &list.Items[i]
		fd, found, err := unstructured.NestedFieldNoCopy(item.Object, "status", "failureDomains")
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if _, ok := fd.([]interface{}); ok {
			continue // already the slice shape
		}

		// Stored as a map (or other non-slice) — reset to an empty slice so the
		// typed List succeeds; the controller repopulates it from spec.
		item.SetGroupVersionKind(maasClusterGVK)
		if err := c.Status().Patch(ctx, item, patch); err != nil {
			return err
		}
		log.Info("migrated MaasCluster status.failureDomains map->slice",
			"namespace", item.GetNamespace(), "name", item.GetName())
	}
	return nil
}
