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

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMigrateMaasClusterFailureDomains(t *testing.T) {
	g := NewGomegaWithT(t)

	s := runtime.NewScheme()
	s.AddKnownTypeWithName(maasClusterGVK, &unstructured.Unstructured{})
	s.AddKnownTypeWithName(maasClusterGVK.GroupVersion().WithKind(maasClusterGVK.Kind+"List"), &unstructured.UnstructuredList{})

	tests := []struct {
		name           string
		failureDomains interface{} // seeded status.failureDomains; nil => absent
		expectPresent  bool        // field present after migration
	}{
		{
			name:           "populated map is converted to slice",
			failureDomains: map[string]interface{}{"az1": map[string]interface{}{"controlPlane": true}},
			expectPresent:  true,
		},
		{
			name:           "empty map is converted to slice",
			failureDomains: map[string]interface{}{},
			expectPresent:  true,
		},
		{
			name:           "already a slice is left as a slice",
			failureDomains: []interface{}{map[string]interface{}{"name": "az1", "controlPlane": true}},
			expectPresent:  true,
		},
		{
			name:           "absent field is left absent",
			failureDomains: nil,
			expectPresent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(maasClusterGVK)
			obj.SetName("c")
			obj.SetNamespace("default")
			if tt.failureDomains != nil {
				g.Expect(unstructured.SetNestedField(obj.Object, tt.failureDomains, "status", "failureDomains")).To(Succeed())
			}

			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(obj).
				WithStatusSubresource(obj).
				Build()

			err := MigrateMaasClusterFailureDomains(context.Background(), c, logr.Discard())
			g.Expect(err).ToNot(HaveOccurred())

			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(maasClusterGVK)
			g.Expect(c.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "c"}, got)).To(Succeed())

			fd, found, err := unstructured.NestedFieldNoCopy(got.Object, "status", "failureDomains")
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(found).To(Equal(tt.expectPresent))
			if tt.expectPresent {
				_, isSlice := fd.([]interface{})
				g.Expect(isSlice).To(BeTrue(), "status.failureDomains must be a slice after migration")
			}
		})
	}
}
