package scope

import (
	"github.com/onsi/gomega"
	infrav1 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/klogr"
	"sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewCluster(t *testing.T) {
	cluster := &v1alpha3.Cluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1alpha3.ClusterSpec{},
		Status:     v1alpha3.ClusterStatus{},
	}

	maasCluster := &infrav1.MaasCluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       infrav1.MaasClusterSpec{},
		Status:     infrav1.MaasClusterStatus{},
	}

	t.Run("new cluster scope", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		scheme := runtime.NewScheme()
		client := fake.NewFakeClientWithScheme(scheme)


		log := klogr.New()
		scope, err := NewClusterScope(ClusterScopeParams{
			Client:              client,
			Logger:              log,
			Cluster:             cluster,
			MaasCluster: maasCluster,
		})

		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(scope).ToNot(gomega.BeNil())


	})
}
