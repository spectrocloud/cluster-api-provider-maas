package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

)

func TestGetDNSResources(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_API_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()
	res, err := c.GetDNSResources(ctx, nil)

	assert.Nil(t, err, "expecting nil error")
	assert.NotNil(t, res, "expecting non-nil result")

	//assert.Equal(t, 1, res.Count, "expecting 1 resource")
	//assert.Equal(t, 1, res.PagesCount, "expecting 1 PAGE found")
	//
	//assert.Equal(t, "integration_face_id", res.Faces[0].FaceID, "expecting correct face_id")
	//assert.NotEmpty(t, res.Faces[0].FaceToken, "expecting non-empty face_token")
	//assert.Greater(t, len(res.Faces[0].FaceImages), 0, "expecting non-empty face_images")
}