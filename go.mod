module github.com/spectrocloud/cluster-api-provider-maas

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.10.0
	github.com/spectrocloud/maas-client-go v0.0.0-20210714065108-3d1cbf51760c
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210709001253-0e1f9d693477
	sigs.k8s.io/cluster-api v0.3.14
	sigs.k8s.io/controller-runtime v0.5.14
)

// replace sigs.k8s.io/cluster-api => ../cluster-api
