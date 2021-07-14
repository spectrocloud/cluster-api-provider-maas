package scope

import (
	"github.com/spectrocloud/maas-client-go/maasclient"
	"os"
)

// NewMaasClient creates a new MaaS client for a given session
// TODO (looking up on Env really the besT? though it is kind of what EC2 does
func NewMaasClient(_ *ClusterScope) *maasclient.Client {

	maasEndpoint := os.Getenv("MAAS_ENDPOINT")
	if maasEndpoint == "" {
		panic("missing env MAAS_ENDPOINT; e.g: MAAS_ENDPOINT=http://10.11.130.10:5240/MAAS")
	}

	maasAPIKey := os.Getenv("MAAS_API_KEY")
	if maasAPIKey == "" {
		panic("missing env MAAS_API_KEY; e.g: MAAS_API_KEY=x:y:z>")
	}

	maasClient := maasclient.NewClient(maasEndpoint, maasAPIKey)
	return maasClient
}
