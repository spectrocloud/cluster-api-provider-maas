package scope

import (
	"github.com/onsi/gomega"
	"os"
	"testing"
)

func TestNewMaasClient(t *testing.T) {
	t.Run("no MAAS env should panic", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)

		f := func() {
			NewMaasClient(&ClusterScope{})
		}

		g.Expect(f).Should(gomega.Panic())
	})

	t.Run("no MAAS key should panic", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)

		os.Setenv("MAAS_ENDPOINT", "http://example.com/MAAS")
		f := func() {
			NewMaasClient(&ClusterScope{})
		}

		g.Expect(f).Should(gomega.Panic())
	})

	t.Run("no MAAS key should panic", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)

		os.Setenv("MAAS_ENDPOINT", "http://example.com/MAAS")
		os.Setenv("MAAS_API_KEY", "a:b:c")

		client := NewMaasClient(&ClusterScope{})
		g.Expect(client).ToNot(gomega.BeNil())
	})
}
