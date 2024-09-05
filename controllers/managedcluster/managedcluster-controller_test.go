package managedcluster

import (
	"context"
	"testing"
	"time"

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
		Client: clientfake.NewClientBuilder().WithScheme(testScheme).Build(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedClusterReconciler"),
		Scheme: testScheme,
	}

	if err := c.Client.Create(ctx, &hivev1.ClusterClaim{
		// TypeMeta: metav1.TypeMeta{
		// 	Kind:       "ClusterClaim",
		// 	APIVersion: hivev1.SchemeGroupVersion.String(),
		// },
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

	if _, err := c.Reconcile(ctx, ctrl.Request{
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

	// the crd.king and api version is not appearing in the annotation
	expectedAnnotation := "test.testns"
	if cluster.Annotations["cluster.open-cluster-management.io/provisioner"] != expectedAnnotation {
		t.Errorf("unexpected annotation, expected: %v, got: %v", expectedAnnotation, cluster.Annotations["cluster.open-cluster-management.io/provisioner"])
	}
	c.SetupWithManager(nil)
}

func TestReconcileClusterDeleteing(t *testing.T) {
	ctx := context.Background()

	c := &ManagedClusterReconciler{
		Client: clientfake.NewClientBuilder().WithScheme(testScheme).Build(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedClusterReconciler"),
		Scheme: testScheme,
	}

	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Client.Create(ctx, &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			DeletionTimestamp: &metav1.Time{
				Time: time.Now(),
			},
		},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileCDDeleteing(t *testing.T) {
	ctx := context.Background()

	c := &ManagedClusterReconciler{
		Client: clientfake.NewClientBuilder().WithScheme(testScheme).Build(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedClusterReconciler"),
		Scheme: testScheme,
	}

	if err := c.Client.Create(ctx, &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Client.Create(ctx, &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec:       hivev1.ClusterDeploymentSpec{},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileCLDeleteing(t *testing.T) {
	ctx := context.Background()

	c := &ManagedClusterReconciler{
		Client: clientfake.NewClientBuilder().WithScheme(testScheme).Build(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedClusterReconciler"),
		Scheme: testScheme,
	}

	if err := c.Client.Create(ctx, &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
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

	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Client.Create(ctx, &hivev1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "testns"},
		Spec:       hivev1.ClusterClaimSpec{},
	}, &client.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "test",
		},
	}); err != nil {
		t.Fatal(err)
	}
}
