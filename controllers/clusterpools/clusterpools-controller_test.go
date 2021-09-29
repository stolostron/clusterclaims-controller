package clusterpools

import (
	"context"
	"errors"
	"testing"
	"time"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/aws"
	"github.com/openshift/hive/apis/hive/v1/azure"
	"github.com/openshift/hive/apis/hive/v1/gcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const CP_NAME = "chlorine-and-salt"
const CP_NAMESPACE = "my-pools"
const CLUSTER01 = "cluster01"
const NO_CLUSTER = ""

var s = scheme.Scheme

func init() {
	corev1.SchemeBuilder.AddToScheme(s)
	hivev1.SchemeBuilder.AddToScheme(s)
}

func getRequest() ctrl.Request {
	return getRequestWithNamespaceName(CP_NAMESPACE, CP_NAME)
}

func getRequestWithNamespaceName(rNamespace string, rName string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: getNamespaceName(rNamespace, rName),
	}
}

func getNamespaceName(namespace string, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func getSecret(namespace string, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{},
	}
}

func GetClusterPoolsReconciler() *ClusterPoolsReconciler {

	// Log levels: DebugLevel  DebugLevel
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel)))

	return &ClusterPoolsReconciler{
		Client: clientfake.NewFakeClientWithScheme(s),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterPoolsReconciler"),
		Scheme: s,
	}
}

func GetClusterPool(namespace string, name string, poolType string) *hivev1.ClusterPool {
	cp := &hivev1.ClusterPool{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hivev1.ClusterPoolSpec{
			PullSecretRef:                  &corev1.LocalObjectReference{Name: "secret01"},
			InstallConfigSecretTemplateRef: &corev1.LocalObjectReference{Name: "secret02"},
			Platform: hivev1.Platform{
				AWS:   nil,
				GCP:   nil,
				Azure: nil,
			},
		},
	}

	switch poolType {
	case "aws":
		cp.Spec.Platform.AWS = &aws.Platform{CredentialsSecretRef: corev1.LocalObjectReference{Name: "secret03"}}
	case "gcp":
		cp.Spec.Platform.GCP = &gcp.Platform{CredentialsSecretRef: corev1.LocalObjectReference{Name: "secret03"}}
	case "azure":
		cp.Spec.Platform.Azure = &azure.Platform{CredentialsSecretRef: corev1.LocalObjectReference{Name: "secret03"}}
	default:
		panic(errors.New("GetClusterPool: Invalid poolType: " + poolType))
	}

	return cp
}

func GetClusterPoolNoRefs(namespace string, name string, poolType string) *hivev1.ClusterPool {
	cp := &hivev1.ClusterPool{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hivev1.ClusterPoolSpec{},
	}

	return cp
}

func TestReconcileClusterPoolsAwsNoSecret(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterPoolsReconciler()

	ccr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME, "aws"), &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")
}

func TestReconcileClusterPoolsGcpNoSecret(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterPoolsReconciler()

	ccr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME, "gcp"), &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")
}

func TestReconcileClusterPoolsAazureNoSecret(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterPoolsReconciler()

	ccr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME, "azure"), &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")
}

func TestReconcileClusterPoolDeleteAws(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "aws")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")
}

func TestReconcileClusterPoolDeleteGcp(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "gcp")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")
}

func TestReconcileClusterPoolDeleteAzure(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "azure")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.NotNil(t, err, "not nil, when secret was successfully deleted")
	assert.Contains(t, err.Error(), " not found", "secret should not be found")
}

func TestReconcileClusterPoolsMissing(t *testing.T) {

	ccr := GetClusterPoolsReconciler()

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is not found (considered deleted) reconcile was successful")
}

func TestReconcileClusterPoolDeleteSharedSecretsAws(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "aws")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})
	cpr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME+"02", "aws"), &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")
}

func TestReconcileClusterPoolDeleteSharedSecretsGcp(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "gcp")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})
	cpr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME+"02", "gcp"), &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")
}

func TestReconcileClusterPoolDeleteSharedSecretsAzure(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "azure")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})
	cpr.Client.Create(ctx, GetClusterPool(CP_NAMESPACE, CP_NAME+"02", "azure"), &client.CreateOptions{})

	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret01"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret02"))
	cpr.Client.Create(ctx, getSecret(CP_NAMESPACE, "secret03"))

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var secret corev1.Secret
	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret01"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret02"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")

	err = cpr.Client.Get(ctx, getNamespaceName(CP_NAMESPACE, "secret03"), &secret)
	assert.Nil(t, err, "nil, when secret was not deleted")
}

func TestReconcileClusterPoolsDeleteNamespace(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "azure")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, cp, &client.CreateOptions{})

	ns := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: CP_NAMESPACE, Labels: map[string]string{LABEL_NAMESPACE: CLUSTERPOOLS}}}

	ccr.Client.Create(ctx, &ns)

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CP_NAMESPACE), &ns)
	assert.NotNil(t, err, "not nil, when namespace is deleted")
	assert.Contains(t, err.Error(), " not found", "namespace should not be found")
}

func TestReconcileClusterPoolDeleteMissingSecretsAws(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "aws")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}

func TestReconcileClusterPoolDeleteMissingSecretsGcp(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "gcp")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}

func TestReconcileClusterPoolDeleteMissingSecretsAzure(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPool(CP_NAMESPACE, CP_NAME, "azure")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}

func TestReconcileClusterPoolDeleteMissingSecretRefsAws(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPoolNoRefs(CP_NAMESPACE, CP_NAME, "aws")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}

func TestReconcileClusterPoolDeleteMissingSecretRefsGcp(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPoolNoRefs(CP_NAMESPACE, CP_NAME, "gcp")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}

func TestReconcileClusterPoolDeleteMissingSecretRefsAzure(t *testing.T) {

	ctx := context.Background()

	cpr := GetClusterPoolsReconciler()

	cp := GetClusterPoolNoRefs(CP_NAMESPACE, CP_NAME, "azure")
	cp.DeletionTimestamp = &v1.Time{time.Now()}

	cpr.Client.Create(ctx, cp, &client.CreateOptions{})

	_, err := cpr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterPool delete reconcile successful")
}
