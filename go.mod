module github.com/jnpacker/clusterclaims-controller

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/open-cluster-management/api v0.0.0-20210527013639-a6845f2ebcb1
	github.com/open-cluster-management/klusterlet-addon-controller v0.0.0-20210624203246-085d806736ce
	github.com/openshift/hive/apis v0.0.0-20210624144808-697460baf215
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.17.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
	k8s.io/api => k8s.io/api v0.20.0
	k8s.io/client-go => k8s.io/client-go v0.20.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.2
)
