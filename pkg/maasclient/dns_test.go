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

	t.Run("get cluster1.maas", func(t *testing.T) {
		options := &GetDNSResourcesOptions{
			FQDN: pointer.StringPtr("cluster1.maas"),
		}
		res, err := c.GetDNSResources(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotEmpty(t, res)
		assert.NotZero(t, res[0].AddressTTL)
		assert.NotEmpty(t, res[0].IpAddresses)
		assert.NotEmpty(t, res[0].IpAddresses[0].IpAddress)

		// TODO create test DNS

	})

	t.Run("create test-unit1.maas", func(t *testing.T) {
		options := CreateDNSResourcesOptions{
			FQDN:        "test-unit1.maas",
			AddressTTL:  "10",
			IpAddresses: []string{},
		}
		res, err := c.CreateDNSResources(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res)
		assert.Equal(t, res.FQDN, "test-unit1.maas")
		assert.Equal(t, *res.AddressTTL, 10)
		assert.Empty(t, res.IpAddresses)

		err = c.DeleteDNSResources(ctx, res.ID)
		assert.Nil(t, err, "expecting nil error")

	})

	t.Run("create test-unit1.maas", func(t *testing.T) {
		options := CreateDNSResourcesOptions{
			FQDN:        "test-unit1.maas",
			AddressTTL:  "10",
			IpAddresses: []string{},
		}
		res, err := c.CreateDNSResources(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res)

		updateOptions := UpdateDNSResourcesOptions{
			ID:          res.ID,
			IpAddresses: []string{"1.2.3.4", "5.6.7.8"},
		}
		res, err = c.UpdateDNSResources(ctx, updateOptions)
		if err != nil {
			t.Fatal("error", err)
		}
		assert.Equal(t, res.FQDN, "test-unit1.maas")
		assert.Equal(t, *res.AddressTTL, 10)
		assert.NotEmpty(t, res.IpAddresses)

		err = c.DeleteDNSResources(ctx, res.ID)
		assert.Nil(t, err, "expecting nil error")

	})

	//assert.Equal(t, 1, res.Count, "expecting 1 resource")

	//assert.Equal(t, 1, res.PagesCount, "expecting 1 PAGE found")
	//
	//assert.Equal(t, "integration_face_id", res.Faces[0].FaceID, "expecting correct face_id")
	//assert.NotEmpty(t, res.Faces[0].FaceToken, "expecting non-empty face_token")
	//assert.Greater(t, len(res.Faces[0].FaceImages), 0, "expecting non-empty face_images")
}
