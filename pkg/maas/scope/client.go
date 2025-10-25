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

package scope

import (
	"context"
	"fmt"
	"os"

	"github.com/spectrocloud/maas-client-go/maasclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewMaasClient creates a new MaaS client for a given session
// TODO (looking up on Env really the besT? though it is kind of what EC2 does
func NewMaasClient(_ *ClusterScope) maasclient.ClientSetInterface {

	maasEndpoint := os.Getenv("MAAS_ENDPOINT")
	if maasEndpoint == "" {
		panic("missing env MAAS_ENDPOINT; e.g: MAAS_ENDPOINT=http://10.11.130.11:5240/MAAS")
	}

	maasAPIKey := os.Getenv("MAAS_API_KEY")
	if maasAPIKey == "" {
		panic("missing env MAAS_API_KEY; e.g: MAAS_API_KEY=x:y:z>")
	}

	maasClient := maasclient.NewAuthenticatedClientSet(maasEndpoint, maasAPIKey)
	return maasClient
}

// NewMaasClientFromSecret constructs a MAAS client using a Secret in the given namespace.
// Secret is expected to contain keys: "endpoint" and "apiKey".
func NewMaasClientFromSecret(ctx context.Context, c client.Client, namespace, secretName string) (maasclient.ClientSetInterface, error) {
	key := types.NamespacedName{Namespace: namespace, Name: secretName}
	var sec corev1.Secret
	if err := c.Get(ctx, key, &sec); err != nil {
		return nil, err
	}
	endpoint := string(sec.Data["endpoint"])
	apiKey := string(sec.Data["apiKey"])
	if endpoint == "" || apiKey == "" {
		return nil, fmt.Errorf("maas secret %s/%s missing endpoint or apiKey", namespace, secretName)
	}
	return maasclient.NewAuthenticatedClientSet(endpoint, apiKey), nil
}
