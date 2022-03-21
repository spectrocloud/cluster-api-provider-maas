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

package v1beta1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMaasCluster_ValidateCreate(t *testing.T) {
	tests := []struct {
		name      string
		dnsDomain string
		wantError bool
	}{
		{
			name:      "should allow creation with dns name",
			dnsDomain: "maas.sc",
			wantError: false,
		},
		{
			name:      "should not allow creation without dns name",
			dnsDomain: "",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &MaasCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       "MaasCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: MaasClusterSpec{
					DNSDomain: tt.dnsDomain,
				},
			}

			ctx := context.TODO()
			if err := testEnv.Create(ctx, cluster); (err != nil) != tt.wantError {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantError)
			}
			testEnv.Delete(ctx, cluster)
		})
	}
}

func TestMAASCluster_Update(t *testing.T) {
	tests := []struct {
		name       string
		oldCluster *MaasCluster
		newCluster *MaasCluster
		wantErr    bool
	}{
		{
			name: "change in dnsDomain should not be allowed",
			oldCluster: &MaasCluster{
				Spec: MaasClusterSpec{
					DNSDomain: "maas.sc",
				},
			},
			newCluster: &MaasCluster{
				Spec: MaasClusterSpec{
					DNSDomain: "maas.maas",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		ctx := context.TODO()
		t.Run(tt.name, func(t *testing.T) {
			cluster := tt.oldCluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "cluster-",
				Namespace:    "default",
			}
			if err := testEnv.Create(ctx, cluster); err != nil {
				t.Errorf("failed to create cluster: %v", err)
			}
			cluster.Spec = tt.newCluster.Spec
			if err := testEnv.Update(ctx, cluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
