package maasclient

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	code := m.Run()
	os.Exit(code)
}

func TestClient_GetMachine(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()
	res, err := c.GetMachine(ctx, "e37xxm")

	assert.Nil(t, err, "expecting nil error")

	assert.NotNil(t, res, "expecting non-nil result")
	assert.NotEmpty(t, res.SystemID)
	assert.NotEmpty(t, res.Hostname)
	assert.Equal(t, res.State, "Deployed")
	assert.NotEmpty(t, res.PowerState)
	assert.Equal(t, res.AvailabilityZone, "az1")

	assert.NotEmpty(t, res.FQDN)
	assert.NotEmpty(t, res.IpAddresses)

	assert.NotEmpty(t, res.OSSystem)
	assert.NotEmpty(t, res.DistroSeries)

	assert.Zero(t, *res.SwapSize)

}

func TestClient_AllocateMachine(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()

	releaseMachine := func(res *Machine) {
		if res != nil {
			err := c.ReleaseMachine(ctx, res.SystemID)
			assert.Nil(t, err)
		}
	}

	t.Run("no-options", func(t *testing.T) {
		res, err := c.AllocateMachine(ctx, nil)

		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res)

		releaseMachine(res)
	})

	t.Run("bad-options", func(t *testing.T) {
		res, err := c.AllocateMachine(ctx, &AllocateMachineOptions{SystemID: pointer.StringPtr("abc")})

		assert.NotNil(t, err, "expecting error")

		releaseMachine(res)
	})

	t.Run("with-az", func(t *testing.T) {
		res, err := c.AllocateMachine(ctx, &AllocateMachineOptions{AvailabilityZone: pointer.StringPtr("az1")})

		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res)

		releaseMachine(res)
	})

}

func TestClient_DeployMachine(t *testing.T) {
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	ctx := context.Background()

	releaseMachine := func(res *Machine) {
		if res != nil {
			err := c.ReleaseMachine(ctx, res.SystemID)
			assert.Nil(t, err)
		}
	}

	t.Run("simple", func(t *testing.T) {
		res, err := c.AllocateMachine(ctx, nil)
		if err != nil {
			t.Fatal("Machine didn't allocate")
		}
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.SystemID)

		options := DeployMachineOptions{
			SystemID:     res.SystemID,
			OSSystem:     pointer.StringPtr("custom"),
			DistroSeries: pointer.StringPtr("spectro-u18-k11815"),
		}

		res, err = c.DeployMachine(ctx, options)
		assert.Nil(t, err, "expecting nil error")
		assert.NotNil(t, res)

		assert.Equal(t, res.OSSystem, "custom")
		assert.Equal(t, res.DistroSeries, "spectro-u18-k11815")

		// Give me a few seconds before clenaing up
		time.Sleep(15 * time.Second)

		releaseMachine(res)
	})

}

func TestClient_UpdateMachine(t *testing.T) {
	ctx := context.Background()
	c := NewClient(os.Getenv("MAAS_ENDPOINT"), os.Getenv("MAAS_API_KEY"))

	swapSize := 0
	options := UpdateMachineOptions{
		SystemID: "e37xxm",
		SwapSize: &swapSize,
	}

	res, err := c.UpdateMachine(ctx, options)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, *res.SwapSize, 0)

}
