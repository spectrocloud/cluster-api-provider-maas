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
	"testing"

	"github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	infrav1beta1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
)

func boolPtr(b bool) *bool { return &b }

// TestIsHCPMaasCluster verifies the HCP discriminator used to route maintenance
// controllers: a MaasCluster is HCP only when spec.lxdConfig.enabled is true.
func TestIsHCPMaasCluster(t *testing.T) {
	tests := []struct {
		name    string
		cluster *infrav1beta1.MaasCluster
		want    bool
	}{
		{
			name:    "nil cluster",
			cluster: nil,
			want:    false,
		},
		{
			name:    "no lxdConfig (standard or WLC cluster)",
			cluster: &infrav1beta1.MaasCluster{},
			want:    false,
		},
		{
			name: "lxdConfig present but enabled nil",
			cluster: &infrav1beta1.MaasCluster{
				Spec: infrav1beta1.MaasClusterSpec{LXDConfig: &infrav1beta1.LXDConfig{}},
			},
			want: false,
		},
		{
			name: "lxdConfig enabled=false",
			cluster: &infrav1beta1.MaasCluster{
				Spec: infrav1beta1.MaasClusterSpec{LXDConfig: &infrav1beta1.LXDConfig{Enabled: boolPtr(false)}},
			},
			want: false,
		},
		{
			name: "lxdConfig enabled=true (HCP cluster)",
			cluster: &infrav1beta1.MaasCluster{
				Spec: infrav1beta1.MaasClusterSpec{LXDConfig: &infrav1beta1.LXDConfig{Enabled: boolPtr(true)}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			g.Expect(isHCPMaasCluster(tt.cluster)).To(gomega.Equal(tt.want))
		})
	}
}

// TestNotHCPClusterPredicate verifies VEC's watch predicate: it accepts non-HCP
// MaasClusters (workload/standard) and rejects HCP clusters and non-MaasCluster
// objects.
func TestNotHCPClusterPredicate(t *testing.T) {
	pred := notHCPClusterPredicate()

	tests := []struct {
		name   string
		obj    client.Object
		accept bool
	}{
		{
			name:   "WLC/standard MaasCluster is accepted",
			obj:    &infrav1beta1.MaasCluster{},
			accept: true,
		},
		{
			name: "HCP MaasCluster is rejected",
			obj: &infrav1beta1.MaasCluster{
				Spec: infrav1beta1.MaasClusterSpec{LXDConfig: &infrav1beta1.LXDConfig{Enabled: boolPtr(true)}},
			},
			accept: false,
		},
		{
			name:   "non-MaasCluster object is rejected",
			obj:    &infrav1beta1.MaasMachine{},
			accept: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			// All of the predicate's Create/Update/Delete handlers delegate to the
			// same func, so asserting Create is sufficient.
			g.Expect(pred.Create(event.CreateEvent{Object: tt.obj})).To(gomega.Equal(tt.accept))
		})
	}
}

// lxdTemplate returns a MaasMachineTemplate with the given LXD config for testing.
func lxdTemplate(lxd *infrav1beta1.MachineLXDConfig) *infrav1beta1.MaasMachineTemplate {
	tmpl := &infrav1beta1.MaasMachineTemplate{}
	tmpl.Spec.Template.Spec.LXD = lxd
	return tmpl
}

// TestTemplateHasLXDEnabled verifies the WLC discriminator: a control-plane
// MaasMachineTemplate is an LXD (workload-cluster) template only when
// spec.template.spec.lxd.enabled is true.
func TestTemplateHasLXDEnabled(t *testing.T) {
	tests := []struct {
		name string
		tmpl *infrav1beta1.MaasMachineTemplate
		want bool
	}{
		{
			name: "nil template",
			tmpl: nil,
			want: false,
		},
		{
			name: "no lxd config (standard/bare-metal)",
			tmpl: lxdTemplate(nil),
			want: false,
		},
		{
			name: "lxd present but enabled nil",
			tmpl: lxdTemplate(&infrav1beta1.MachineLXDConfig{}),
			want: false,
		},
		{
			name: "lxd enabled=false",
			tmpl: lxdTemplate(&infrav1beta1.MachineLXDConfig{Enabled: boolPtr(false)}),
			want: false,
		},
		{
			name: "lxd enabled=true (WLC)",
			tmpl: lxdTemplate(&infrav1beta1.MachineLXDConfig{Enabled: boolPtr(true)}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			g.Expect(templateHasLXDEnabled(tt.tmpl)).To(gomega.Equal(tt.want))
		})
	}
}

// newKCP builds an unstructured KubeadmControlPlane referencing the given
// control-plane MaasMachineTemplate name in namespace ns.
func newKCP(name, ns, templateName string) *unstructured.Unstructured {
	kcp := &unstructured.Unstructured{}
	kcp.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "controlplane.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "KubeadmControlPlane",
	})
	kcp.SetName(name)
	kcp.SetNamespace(ns)
	if templateName != "" {
		// v1beta2 contract: infrastructureRef lives under spec.machineTemplate.spec
		// and is a ContractVersionedObjectReference (no namespace — same as the KCP).
		_ = unstructured.SetNestedMap(kcp.Object, map[string]interface{}{
			"name": templateName,
		}, "spec", "machineTemplate", "spec", "infrastructureRef")
	}
	return kcp
}

// kcpCRD returns the KubeadmControlPlane CRD carrying the CAPI contract-version
// label, which external.GetObjectFromContractVersionedRef reads to resolve the
// served apiVersion of a contract-versioned controlPlaneRef.
func kcpCRD() client.Object {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeadmcontrolplanes.controlplane.cluster.x-k8s.io",
			Labels: map[string]string{
				clusterv1.GroupVersion.Group + "/v1beta2": "v1beta2",
			},
		},
	}
}

// TestIsWLCCluster verifies VEC's cluster-type gate: a cluster is a WLC only when
// its control-plane MaasMachineTemplate has lxd.enabled=true. Standard clusters
// return false, and an unresolvable KCP returns an error so the caller retries.
func TestIsWLCCluster(t *testing.T) {
	const ns = "default"

	cluster := func() *clusterv1.Cluster {
		return &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns},
			Spec: clusterv1.ClusterSpec{
				ControlPlaneRef: clusterv1.ContractVersionedObjectReference{
					APIGroup: "controlplane.cluster.x-k8s.io",
					Kind:     "KubeadmControlPlane",
					Name:     "kcp",
				},
			},
		}
	}

	tests := []struct {
		name      string
		objs      []client.Object
		unstructs []*unstructured.Unstructured
		wantWLC   bool
		wantErr   bool
	}{
		{
			name: "WLC: CP template lxd.enabled=true",
			objs: []client.Object{
				kcpCRD(),
				func() client.Object {
					tmpl := lxdTemplate(&infrav1beta1.MachineLXDConfig{Enabled: boolPtr(true)})
					tmpl.Name, tmpl.Namespace = "cp-template", ns
					return tmpl
				}(),
			},
			unstructs: []*unstructured.Unstructured{newKCP("kcp", ns, "cp-template")},
			wantWLC:   true,
			wantErr:   false,
		},
		{
			name: "standard: CP template lxd disabled",
			objs: []client.Object{
				kcpCRD(),
				func() client.Object {
					tmpl := lxdTemplate(&infrav1beta1.MachineLXDConfig{Enabled: boolPtr(false)})
					tmpl.Name, tmpl.Namespace = "cp-template", ns
					return tmpl
				}(),
			},
			unstructs: []*unstructured.Unstructured{newKCP("kcp", ns, "cp-template")},
			wantWLC:   false,
			wantErr:   false,
		},
		{
			name:      "KCP missing: error so caller retries",
			objs:      nil,
			unstructs: nil,
			wantWLC:   false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			scheme := runtime.NewScheme()
			g.Expect(infrav1beta1.AddToScheme(scheme)).To(gomega.Succeed())
			g.Expect(clusterv1.AddToScheme(scheme)).To(gomega.Succeed())
			g.Expect(apiextensionsv1.AddToScheme(scheme)).To(gomega.Succeed())

			builder := fake.NewClientBuilder().WithScheme(scheme)
			if len(tt.objs) > 0 {
				builder = builder.WithObjects(tt.objs...)
			}
			for _, u := range tt.unstructs {
				builder = builder.WithObjects(u)
			}

			r := &VMEvacuationReconciler{Client: builder.Build(), Log: klogr.New(), Scheme: scheme}

			gotWLC, err := r.isWLCCluster(context.Background(), cluster())
			if tt.wantErr {
				g.Expect(err).To(gomega.HaveOccurred())
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
			}
			g.Expect(gotWLC).To(gomega.Equal(tt.wantWLC))
		})
	}
}
