module github.com/spectrocloud/cluster-api-provider-maas

go 1.16

require (
	github.com/go-logr/logr v1.2.0
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.32.1
	github.com/spectrocloud/maas-client-go v0.0.1-beta1.0.20210805102600-28f250f3bdc7
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.23.0
	k8s.io/apiextensions-apiserver v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	k8s.io/klog/v2 v2.30.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.1.3
	sigs.k8s.io/controller-runtime v0.11.1
)

//github.com/go-logr/logr v1.2.0 => github.com/go-logr/logr v0.4.0
replace github.com/prometheus/common v0.32.1 => github.com/prometheus/common v0.26.0
