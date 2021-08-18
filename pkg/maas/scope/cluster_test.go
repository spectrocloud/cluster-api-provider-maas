package scope

import (
	"testing"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/klogr"
	"sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1alpha4 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
)

func TestNewCluster(t *testing.T) {
	cluster := &v1alpha4.Cluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1alpha4.ClusterSpec{},
		Status:     v1alpha4.ClusterStatus{},
	}

	maasCluster := &infrav1alpha4.MaasCluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       infrav1alpha4.MaasClusterSpec{},
		Status:     infrav1alpha4.MaasClusterStatus{},
	}

	t.Run("new cluster scope", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		scheme := runtime.NewScheme()
		_ = infrav1alpha4.AddToScheme(scheme)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()

		log := klogr.New()
		scope, err := NewClusterScope(ClusterScopeParams{
			Client:      client,
			Logger:      log,
			Cluster:     cluster,
			MaasCluster: maasCluster,
		})

		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(scope).ToNot(gomega.BeNil())

	})

	t.Run("new dns name", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		scheme := runtime.NewScheme()
		_ = infrav1alpha4.AddToScheme(scheme)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		clusterCopy := cluster.DeepCopy()
		clusterCopy.Name = "dns-test"
		maasClusterCopy := maasCluster.DeepCopy()
		maasClusterCopy.Spec.DNSDomain = "maas.com"
		log := klogr.New()
		scope, err := NewClusterScope(ClusterScopeParams{
			Client:      client,
			Logger:      log,
			Cluster:     clusterCopy,
			MaasCluster: maasClusterCopy,
		})

		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(scope.GetDNSName()).ToNot(gomega.BeNil())
		g.Expect(scope.GetDNSName()).To(gomega.ContainSubstring(clusterCopy.Name))
		g.Expect(scope.GetDNSName()).To(gomega.ContainSubstring(maasClusterCopy.Spec.DNSDomain))
		dnsLengh := len("dns-test-") + DnsSuffixLength + len(".maas.com")
		g.Expect(len(scope.GetDNSName())).To(gomega.Equal(dnsLengh))
	})
}
