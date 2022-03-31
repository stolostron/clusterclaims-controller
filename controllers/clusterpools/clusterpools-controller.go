// Copyright Contributors to the Open Cluster Management project.

package clusterpools

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const DEBUG = 1
const INFO = 0
const WARN = -1
const ERROR = -2
const FINALIZER = "clusterpools-controller.open-cluster-management.io/cleanup"

const LABEL_NAMESPACE = "open-cluster-management.io/managed-by"
const CLUSTERPOOLS = "clusterpools"

// ClusterPoolsReconciler reconciles a ClusterPool, mainly for the delete
type ClusterPoolsReconciler struct {
	KubeClient kubernetes.Interface
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ClusterPoolsReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()

	log := r.Log.WithValues("ClusterPoolsReconciler", req.NamespacedName)

	var cp hivev1.ClusterPool
	if err := r.Get(ctx, req.NamespacedName, &cp); err != nil {
		log.V(INFO).Info("Resource deleted")

		return ctrl.Result{}, nil
	}

	// Early exit
	if cp.DeletionTimestamp == nil && controllerutil.ContainsFinalizer(&cp, FINALIZER) {
		return ctrl.Result{}, nil
	}

	target := cp.Name
	log.V(INFO).Info("Reconcile cluster pool: " + target)

	if cp.DeletionTimestamp != nil {
		if err := deleteResources(r, &cp); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, removeFinalizer(r, &cp)
	}

	return ctrl.Result{}, setFinalizer(r, &cp)
}

func (r *ClusterPoolsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hivev1.ClusterPool{}).WithEventFilter(predicate.Funcs{
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

func setFinalizer(r *ClusterPoolsReconciler, cc *hivev1.ClusterPool) error {

	patch := client.MergeFrom(cc.DeepCopy())

	controllerutil.AddFinalizer(cc, FINALIZER)

	return r.Patch(context.Background(), cc, patch)
}

func removeFinalizer(r *ClusterPoolsReconciler, cc *hivev1.ClusterPool) error {

	if !controllerutil.ContainsFinalizer(cc, FINALIZER) {
		return nil
	}

	controllerutil.RemoveFinalizer(cc, FINALIZER)

	err := r.Update(context.Background(), cc)
	if err == nil {
		r.Log.V(INFO).Info("Removed finalizer on cluster pool: " + cc.Name)
	}
	return err

}
func getCPDetails(cp hivev1.ClusterPool) (cpType string, providerSecretName string) {
	if cp.Spec.Platform.AWS != nil {
		return "aws", cp.Spec.Platform.AWS.CredentialsSecretRef.Name
	} else if cp.Spec.Platform.GCP != nil {
		return "gcp", cp.Spec.Platform.GCP.CredentialsSecretRef.Name
	} else if cp.Spec.Platform.Azure != nil {
		return "azure", cp.Spec.Platform.Azure.CredentialsSecretRef.Name
	}
	return "skip", ""
}
func deleteResources(r *ClusterPoolsReconciler, cp *hivev1.ClusterPool) error {
	ctx := context.Background()
	log := r.Log

	var cps hivev1.ClusterPoolList
	if err := r.List(ctx, &cps, &client.ListOptions{Namespace: cp.Namespace}); err != nil {

		if k8serrors.IsNotFound(err) {
			log.V(INFO).Info("No Cluster Pools found")
			return nil
		} else {
			return err
		}

	} else {

		// Remove secrets that are not used by any other cluster pool in the namespace
		foundPullSecret := false
		foundInstallConfigSecret := false
		foundProviderSecret := false

		cpType, providerSecretName := getCPDetails(*cp)

		for _, foundCp := range cps.Items {

			// Skip if the cluster pool being deleted is the element in the list
			if cp.Name == foundCp.Name {
				continue
			}

			if cp.Spec.PullSecretRef != nil && foundCp.Spec.PullSecretRef != nil && cp.Spec.PullSecretRef.Name == foundCp.Spec.PullSecretRef.Name {
				foundPullSecret = true
			}

			if cp.Spec.InstallConfigSecretTemplateRef != nil && foundCp.Spec.InstallConfigSecretTemplateRef != nil && cp.Spec.InstallConfigSecretTemplateRef.Name == foundCp.Spec.InstallConfigSecretTemplateRef.Name {
				foundInstallConfigSecret = true
			}

			// This needs to happen after the cp.Name == foundCp.Name check

			foundCpType, foundProviderSecretName := getCPDetails(foundCp)

			if cpType == foundCpType && providerSecretName == foundProviderSecretName {
				foundProviderSecret = true
			}
		}

		log.V(INFO).Info(
			fmt.Sprintf("Shared secrets found, install-config: %v, Pull secret: %v, Provider credential: %v",
				foundInstallConfigSecret, foundPullSecret, foundProviderSecret))

		log.V(DEBUG).Info(fmt.Sprintf("providerSecretName: %v", providerSecretName))

		if !foundInstallConfigSecret && cp.Spec.InstallConfigSecretTemplateRef != nil {

			if err := deleteSecret(r, cp.Namespace, cp.Spec.InstallConfigSecretTemplateRef.Name); err != nil {
				return err
			}
			log.V(INFO).Info("Deleted install-config secret: " + cp.Spec.InstallConfigSecretTemplateRef.Name)
		}

		if !foundPullSecret && cp.Spec.PullSecretRef != nil {

			if err := deleteSecret(r, cp.Namespace, cp.Spec.PullSecretRef.Name); err != nil {
				return err
			}
			log.V(INFO).Info("Deleted Pull-Secret secret: " + cp.Spec.PullSecretRef.Name)
		}

		if !foundProviderSecret && providerSecretName != "" {

			if err := deleteSecret(r, cp.Namespace, providerSecretName); err != nil {
				return err
			}
			log.V(INFO).Info("Deleted Provider-Credential secret: " + providerSecretName)
		}
	}

	return nil
}

func deleteSecret(r *ClusterPoolsReconciler, namespace string, name string) error {
	ctx := context.Background()
	// Keep going if the secret is not found, but if found, remove it
	_, err := r.KubeClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.V(WARN).Info("Secret: " + name + " was not found")
			return nil
		}
		return err
	}

	return r.KubeClient.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
