package maasclient

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"os"
	"testing"
)

func TestGetDNSResources(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()

	t.Run("no-options", func(t *testing.T) {
		res, err := c.GetDNSResources(ctx, nil)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res, "expecting non-nil result")

		assert.Greater(t, len(res), 0, "expecting non-empty dns_resources")

		assert.NotZero(t, res[0].ID)
		assert.NotEmpty(t, res[0].FQDN)
	})

	t.Run("invalid-search", func(t *testing.T) {
		options := &GetDNSResourcesOptions{
			FQDN: pointer.StringPtr("bad-doesntexist.maas"),
		}
		res, err := c.GetDNSResources(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res, "expecting non-nil result")
		assert.Empty(t, res)

	})

	t.Run("cluster1.maas", func(t *testing.T) {
		options := &GetDNSResourcesOptions{
			FQDN: pointer.StringPtr("cluster1.maas"),
		}
		res, err := c.GetDNSResources(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotEmpty(t, res)
		assert.NotEmpty(t, res[0].IpAddresses)
		assert.NotEmpty(t, res[0].IpAddresses[0].IpAddress)

		// TODO create test DNS

	})

	//assert.Equal(t, 1, res.Count, "expecting 1 resource")

	//assert.Equal(t, 1, res.PagesCount, "expecting 1 PAGE found")
	//
	//assert.Equal(t, "integration_face_id", res.Faces[0].FaceID, "expecting correct face_id")
	//assert.NotEmpty(t, res.Faces[0].FaceToken, "expecting non-empty face_token")
	//assert.Greater(t, len(res.Faces[0].FaceImages), 0, "expecting non-empty face_images")
}
