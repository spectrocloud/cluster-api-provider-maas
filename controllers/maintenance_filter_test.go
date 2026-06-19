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
	"testing"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
