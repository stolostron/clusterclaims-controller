package managedcluster

import (
	"context"
	"fmt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const provisionerAnnotation = "cluster.open-cluster-management.io/provisioner"

// ManagedClusterReconciler reconciles a managed cluster
type ManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ManagedClusterReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	clusterName := request.Name
	cluster := &clusterv1.ManagedCluster{}
	err := r.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
	if errors.IsNotFound(err) {
		// cluster is not found, do nothing
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if !cluster.DeletionTimestamp.IsZero() {
		// cluster is deleting, do nothing
		return ctrl.Result{}, nil
	}

	clusterDeployment := &hivev1.ClusterDeployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: clusterName, Name: clusterName}, clusterDeployment)
	if errors.IsNotFound(err) {
		// clusterdeployment is not found, do nothing
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if clusterDeployment.Spec.ClusterPoolRef == nil {
		// no cluster pool ref, do nothing
		return ctrl.Result{}, nil
	}

	clusterClaim := &hivev1.ClusterClaim{}
	claimName := clusterDeployment.Spec.ClusterPoolRef.ClaimName
	claimNamespace := clusterDeployment.Spec.ClusterPoolRef.Namespace
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: claimNamespace, Name: claimName}, clusterClaim)
	if errors.IsNotFound(err) {
		// clusterclaim is not found, do nothing
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// annotation format is <name>.<namespace>.<kind>.<apiversion>
	expectedProvisioner := fmt.Sprintf("%s.%s.%s.%s", claimName, claimNamespace, clusterClaim.Kind, clusterClaim.APIVersion)

	patch := client.MergeFrom(cluster.DeepCopy())

	annotations := cluster.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if provisioner, ok := annotations[provisionerAnnotation]; !ok || provisioner != expectedProvisioner {
		annotations[provisionerAnnotation] = expectedProvisioner
		cluster.Annotations = annotations
		return ctrl.Result{}, r.Client.Patch(ctx, cluster, patch)
	}

	return ctrl.Result{}, nil
}

func (r *ManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.ManagedCluster{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}).WithOptions(controller.Options{
		MaxConcurrentReconciles: 1, // This is the default
	}).Complete(r)
}
