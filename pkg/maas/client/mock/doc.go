// Run go generate to regenerate this mock.
//go:generate ../../../../hack/tools/bin/mockgen -destination clienset_mock.go -package mock_clientset github.com/spectrocloud/maas-client-go/maasclient ClientSetInterface,Machines,DNSResources,DNSResource,DNSResourceBuilder,DNSResourceModifier,IPAddress
//go:generate /usr/bin/env bash -c "cat ../../../../hack/boilerplate.go.txt ec2api_mock.go > _ec2api_mock.go && mv _ec2api_mock.go ec2api_mock.go"

package mock_clientset
