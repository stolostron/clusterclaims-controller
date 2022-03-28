package clusterlcaims

import (
	"context"
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
	mcv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const CC_NAME = "my-clusterclaim"
const CC_NAMESPACE = "my-pool"
const CP_NAME = "chlorine-and-salt"
const CLUSTER01 = "cluster01"
const NO_CLUSTER = ""

var s = scheme.Scheme

func init() {
	corev1.SchemeBuilder.AddToScheme(s)
	hivev1.SchemeBuilder.AddToScheme(s)
	mcv1.AddToScheme(s)
}

func getRequest() ctrl.Request {
	return getRequestWithNamespaceName(CC_NAMESPACE, CC_NAME)
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

func GetClusterClaim(namespace string, name string, clusterName string) *hivev1.ClusterClaim {
	return &hivev1.ClusterClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"usage": "production",
			},
		},
		Spec: hivev1.ClusterClaimSpec{
			ClusterPoolName: "make-believe",
			Namespace:       clusterName,
		},
	}
}

func GetClusterPool(namespace string, name string, labels map[string]string) *hivev1.ClusterPool {
	return &hivev1.ClusterPool{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: hivev1.ClusterPoolSpec{},
	}
}

func GetClusterDeployment(namespace string, cloudType string) *hivev1.ClusterDeployment {
	cd := hivev1.ClusterDeployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
			Labels: map[string]string{
				"usage": "production",
			},
		},
	}
	switch cloudType {
	case "gcp":
		cd.Spec.Platform = hivev1.Platform{GCP: &gcp.Platform{Region: "europe-west3"}}
	case "aws":
		cd.Spec.Platform = hivev1.Platform{AWS: &aws.Platform{Region: "us-east-1"}}
	case "azure":
		cd.Spec.Platform = hivev1.Platform{Azure: &azure.Platform{Region: "centralus"}}
	}

	return &cd
}

func GetClusterClaimsReconciler() *ClusterClaimsReconciler {

	// Log levels: DebugLevel  DebugLevel
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel)))

	return &ClusterClaimsReconciler{
		Client: clientfake.NewFakeClientWithScheme(s),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterClaimsReconciler"),
		Scheme: s,
	}
}

func TestReconcileClusterClaims(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")
}

func TestReconcileClusterClaimsLabelCopy(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["vendor"], "OpenShift", "label vendor should equal OpenShift")
	assert.Equal(t, mc.Labels["usage"], "production", "label usage should equal production")
}
func TestReconcileExistingManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

}

func TestReconcileDeletedClusterClaim(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.NotNil(t, err, "nil, when managedCluster resource is retrieved")
	assert.Contains(t, err.Error(), " not found", "error should be NotFound")
}

func TestReconcileDeletedClusterClaimWithAlreadyDeletingManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}
	mc.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.Nil(t, err, "nil, when managedCluster resource is skipped because it is already deleting")
}

func TestReconcileSkipCreateManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	// Do not create a ManagedCluster for import
	cc.Annotations = map[string]string{CREATECM: "false"}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}
	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	//Check that the ManagedCluster was not created
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.Contains(t, err.Error(), " not found", "for managedCluster, when createmanagedcluster=false")
}

func TestReconcileDeletedClusterClaimWithFalseCreateManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.DeletionTimestamp = &v1.Time{time.Now()}

	// Do not create a ManagedCluster for import
	cc.Annotations = map[string]string{CREATECM: "false"}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.NotNil(t, err, "nil, when managedCluster resource is retrieved")
	assert.Contains(t, err.Error(), " not found", "error should be NotFound")
}

func TestReconcileClusterClaimsLabelCopyForRegionAws(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})
	ccr.Client.Create(ctx, GetClusterDeployment(CLUSTER01, "aws"), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["region"], "us-east-1", "label region should equal us-east-1")
}

func TestReconcileClusterClaimsLabelCopyForRegionGcp(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})
	ccr.Client.Create(ctx, GetClusterDeployment(CLUSTER01, "gcp"), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["region"], "europe-west3", "label region should equal europe-west3")
}

func TestReconcileClusterClaimsLabelCopyForRegionAzure(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})
	ccr.Client.Create(ctx, GetClusterDeployment(CLUSTER01, "azure"), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["region"], "centralus", "label region should equal centralus")
}

func TestReconcileClusterClaimsWithNoLabel(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.Labels = nil

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})
	ccr.Client.Create(ctx, GetClusterDeployment(CLUSTER01, "azure"), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["region"], "centralus", "label region should equal centralus")
}

func TestReconcileClusterSetLabel(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()
	clusterClaim := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)
	ccr.Client.Create(ctx, clusterClaim, &client.CreateOptions{})

	labels := map[string]string{
		ClusterSetLabel: "s1",
	}
	ccr.Client.Create(ctx, GetClusterPool(CC_NAMESPACE, clusterClaim.Spec.ClusterPoolName, labels), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")
	if mc.Labels[ClusterSetLabel] != labels[ClusterSetLabel] {
		t.Errorf("Failed to sync clusterset label to managedclusters")
	}
}

func TestReconcileClusterClaimsNoReimport(t *testing.T) {

	// Delete the ManagedCluster and make sure it is not recreated
	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	_, err := ccr.Reconcile(ctx, getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	err = ccr.Client.Delete(ctx, &mc)
	assert.Nil(t, err, "nil, when managedCluster resource was deleted")

	// Now reconcile
	_, err = ccr.Reconcile(ctx, getRequest())
	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.NotNil(t, err, "not nil, when managedCluster resource is not recreated")

}
