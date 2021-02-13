module github.com/spectrocloud/cluster-api-provider-maas

go 1.15

require (
	github.com/dghubble/oauth1 v0.7.0
	github.com/go-logr/logr v0.3.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/tools v0.0.0-20200616195046-dc31b401abb5 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/cluster-api v0.3.3
	sigs.k8s.io/controller-runtime v0.8.2
)

replace sigs.k8s.io/cluster-api => ../cluster-api
