package managedcluster

import (
	"context"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = scheme.Scheme

func init() {
	corev1.SchemeBuilder.AddToScheme(testScheme)
	hivev1.SchemeBuilder.AddToScheme(testScheme)
	clusterv1.AddToScheme(testScheme)
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	c := &ManagedClusterReconciler{
		Client: clientfake.NewFakeClientWithScheme(testScheme),
		Scheme: testScheme,
	}

	if err := c.Client.Create(ctx, &hivev1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "testns"},
		Spec:       hivev1.ClusterClaimSpec{},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := c.Client.Create(ctx, &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: hivev1.ClusterDeploymentSpec{
			ClusterPoolRef: &hivev1.ClusterPoolReference{ClaimName: "test", Namespace: "testns"},
		},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := c.Client.Create(ctx, &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Reconcile(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}

	cluster := &clusterv1.ManagedCluster{}
	if err := c.Client.Get(ctx, types.NamespacedName{Name: "test"}, cluster); err != nil {
		t.Fatal(err)
	}

	if cluster.Annotations["cluster.open-cluster-management.io/provisioner"] != "test.testns.ClusterClaim.hive.openshift.io/v1" {
		t.Errorf("unexpected annotation %v", cluster.Annotations["provisioner"])
	}
}
