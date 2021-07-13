package maasclient

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetBootResources(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()

	t.Run("list-all", func(t *testing.T) {
		list, err := c.ListBootResources(ctx)
		assert.Nil(t, err, "expecting nil error")
		assert.NotEmpty(t, list)
	})

	t.Run("list-by-id", func(t *testing.T) {
		resource, err := c.GetBootResource(ctx, "7")
		assert.Nil(t, err)
		assert.NotNil(t, resource)
	})

	t.Run("list-importing", func(t *testing.T) {
		status, err := c.BootResourcesImporting(ctx)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, status)
		assert.False(t, *status)
	})


	t.Run("import image", func(t *testing.T) {
		status, err := c.UploadBootResource(ctx, UploadBootResourceInput{
			Name:         "test-image",
			Architecture: "amd64/generic",
			Digest:       "e9844638c7345d182c5d88e1eaeae74749d02beeca38587a530207fddc0a280a",
			Size:         "1262032476",
			Title:        "dstestimage",
			File:         "/Users/deepak/maas/ubuntu.tar.gz",
		})
		assert.NotNil(t, err)
		assert.NotNil(t, status)
	})
}
