// Run go generate to regenerate this mock.
//go:generate ../../../../hack/tools/bin/mockgen -destination clienset_mock.go -package mock_clientset github.com/cloud104/maas-client-go/maasclient ClientSetInterface,Machines,Machine,DNSResources,DNSResource,DNSResourceBuilder,DNSResourceModifier,IPAddress,Zone,MachineReleaser,MachineAllocator,MachineModifier,MachineDeployer
//go:generate /usr/bin/env bash -c "cat ../../../../hack/boilerplate.go.txt clienset_mock.go > _clienset_mock.go && mv _clienset_mock.go clienset_mock.go"

package mock_clientset
