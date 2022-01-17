// Copyright Contributors to the Open Cluster Management project.

package clusterlcaims

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	kacv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	mcv1 "open-cluster-management.io/api/cluster/v1"
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
const FINALIZER = "clusterclaims-controller.open-cluster-management.io/cleanup"
const CREATECM = "cluster.open-cluster-management.io/createmanagedcluster"
const ClusterSetLabel = "cluster.open-cluster-management.io/clusterset"

// ClusterClaimsReconciler reconciles a clusterClaim
type ClusterClaimsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ClusterClaimsReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()

	log := r.Log.WithValues("ClusterClaimsReconciler", req.NamespacedName)

	var cc hivev1.ClusterClaim
	if err := r.Get(ctx, req.NamespacedName, &cc); err != nil {
		log.V(INFO).Info("Resource deleted")

		return ctrl.Result{}, nil
	}

	target := cc.Spec.Namespace
	if target == "" {
		log.V(WARN).Info("Waiting for cluster claim " + cc.Name + " to complete")

		// Requeue
		return ctrl.Result{}, nil
	}
	log.V(INFO).Info("Reconcile cluster: " + target + " for cluster claim: " + cc.Name)

	// Delete the ManagedCluster and KlusterletAddonConfig
	if cc.DeletionTimestamp != nil {
		if err := deleteResources(r, target); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, removeFinalizer(r, &cc)
	}

	// Get the region for a cloud provider and add it to the cc.Labels
	if err := setRegion(r, &cc); err != nil {
		return ctrl.Result{}, err
	}

	if err := setClusterSetLabel(r, &cc); err != nil {
		return ctrl.Result{}, err
	}

	// Do not exit till this point when importmanagedcluster=false, so deletion will work properly if manually imported
	if len(cc.Annotations) > 0 {
		aValue, found := cc.Annotations[CREATECM]
		if found && strings.ToLower(aValue) == "false" {
			log.V(WARN).Info("Skip creation of managedCluster and KlusterletAddonConfig")
			return ctrl.Result{}, nil
		}
	}

	// ManagedCluster
	if res, err := createManagedCluster(r, cc.Name, target, cc.Labels); err != nil {
		return res, err
	}

	//Make sure we don't create the ManagedCluster if it is detached, uses the finalizer to update cc
	if len(cc.Annotations) > 0 {
		cc.Annotations[CREATECM] = "false"
	} else {
		cc.ObjectMeta.Annotations = map[string]string{CREATECM: "false"}
	}

	if err := setFinalizer(r, &cc); err != nil {
		return ctrl.Result{}, err
	}

	// KlusterletAddonConfig
	return createKlusterletAddonConfig(r, target)
}

func (r *ClusterClaimsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hivev1.ClusterClaim{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}).WithOptions(controller.Options{
		MaxConcurrentReconciles: 1, // This is the default
	}).Complete(r)
}

func createManagedCluster(
	r *ClusterClaimsReconciler,
	claimName string,
	target string,
	labels map[string]string) (ctrl.Result, error) {

	log := r.Log
	ctx := context.Background()

	var mc mcv1.ManagedCluster
	err := r.Get(ctx, types.NamespacedName{Name: target}, &mc)
	if k8serrors.IsNotFound(err) {

		log.V(INFO).Info("Create a new ManagedCluster resource")
		mc.Name = target
		mc.Spec.HubAcceptsClient = true

		// Build the labels
		newLabels := map[string]string{}
		if labels != nil {
			for key, val := range labels {
				log.V(DEBUG).Info("Copy label: " + key)
				newLabels[key] = val
			}
		}

		// Use the ClusterClaim name instead of the actual cluster name if a name was not included from the ClusterClaim
		if _, ok := newLabels["name"]; !ok {
			newLabels["name"] = claimName
		}
		newLabels["vendor"] = "OpenShift"  // This is always true
		newLabels["cloud"] = "auto-detect" //This is used to detect cloud provider, like: GCP,AWS

		// Add region lookup. It is a label on the ClusterDeployment or ClusterPool
		mc.ObjectMeta.Labels = newLabels

		if err = r.Create(ctx, &mc, &client.CreateOptions{}); err != nil {

			log.V(ERROR).Info("Could not create ManagedCluster resource: " + target)
			return ctrl.Result{}, err
		}

	} else if err != nil {

		log.V(WARN).Info("Error when attempting to retreive the ManagedCluster resource: " + target)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func createKlusterletAddonConfig(r *ClusterClaimsReconciler, target string) (ctrl.Result, error) {
	log := r.Log
	ctx := context.Background()

	var kac kacv1.KlusterletAddonConfig
	err := r.Get(ctx, types.NamespacedName{Namespace: target, Name: target}, &kac)
	if k8serrors.IsNotFound(err) {

		log.V(INFO).Info("Create a new KlusterletAddonConfig resource")
		kac.Name = target
		kac.Namespace = target
		kac.Spec.ClusterName = target
		kac.Spec.ClusterNamespace = target
		kac.Spec.ClusterLabels = map[string]string{"vendor": "OpenShift"} // Required for object to be created
		kac.Spec.ApplicationManagerConfig.Enabled = true
		kac.Spec.CertPolicyControllerConfig.Enabled = true
		kac.Spec.IAMPolicyControllerConfig.Enabled = true
		kac.Spec.PolicyController.Enabled = true
		kac.Spec.SearchCollectorConfig.Enabled = true

		if err = r.Create(ctx, &kac, &client.CreateOptions{}); err != nil {

			log.V(ERROR).Info("Could not create KlusterletAddonConfig resource: " + target)
			return ctrl.Result{}, err
		}

	} else if err != nil {

		log.V(WARN).Info("Error when attempting to retreive the KlusterletAddonConfig resource: " + target)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

func setFinalizer(r *ClusterClaimsReconciler, cc *hivev1.ClusterClaim) error {

	controllerutil.AddFinalizer(cc, FINALIZER)

	return r.Update(context.Background(), cc)
}

func removeFinalizer(r *ClusterClaimsReconciler, cc *hivev1.ClusterClaim) error {

	if !controllerutil.ContainsFinalizer(cc, FINALIZER) {
		return nil
	}

	controllerutil.RemoveFinalizer(cc, FINALIZER)

	r.Log.V(INFO).Info("Removed finalizer on cluster claim: " + cc.Name)
	return r.Update(context.Background(), cc)

}

func deleteResources(r *ClusterClaimsReconciler, target string) error {
	ctx := context.Background()
	log := r.Log

	var mc mcv1.ManagedCluster
	if err := r.Get(ctx, types.NamespacedName{Name: target}, &mc); err != nil {

		if k8serrors.IsNotFound(err) {
			log.V(WARN).Info("The ManagedCluster resource: " + target + " was not found, can not delete")
		} else {
			return err
		}

	} else {

		if mc.DeletionTimestamp == nil {

			err = r.Delete(ctx, &mc)
			if err != nil {
				log.V(WARN).Info("Error while deleting ManagedCluster resource: " + target)
			}
			log.V(INFO).Info("Deleted ManagedCluster resource: " + target)

		} else {
			log.V(WARN).Info("The managedCluster resource: " + target + " is already being deleted")
		}
	}

	var kac kacv1.KlusterletAddonConfig
	if err := r.Get(ctx, types.NamespacedName{Namespace: target, Name: target}, &kac); err != nil {

		if k8serrors.IsNotFound(err) {
			log.V(WARN).Info("The KlusterletAddonConfig resource: " + target + " was not found, can not delete")
		} else {
			return err
		}

	} else {

		if kac.DeletionTimestamp == nil {

			err = r.Delete(ctx, &kac)
			if err != nil {
				log.V(WARN).Info("Error while deleting KlusterletAddonConfig resource: " + target)
			}
			log.V(INFO).Info("Deleted KlusterletAddonConfig resource: " + target)

		} else {
			log.V(WARN).Info("The klusterletAddonConfig resource: " + target + " is already being deleted")
		}
	}
	return nil
}

func setRegion(r *ClusterClaimsReconciler, cc *hivev1.ClusterClaim) error {

	clusterName := cc.Spec.Namespace
	var cd hivev1.ClusterDeployment
	log := r.Log

	if err := r.Client.Get(
		context.Background(),
		types.NamespacedName{Namespace: clusterName,
			Name: clusterName}, &cd); err != nil {

		if k8serrors.IsNotFound(err) {
			r.Log.V(WARN).Info("No ClusterDeployment found for " + clusterName)
		} else {
			return err
		}

	}

	if cc.Labels == nil {
		cc.Labels = make(map[string]string)
	}

	switch {
	case cd.Spec.Platform.AWS != nil:
		cc.Labels["region"] = cd.Spec.Platform.AWS.Region
	case cd.Spec.Platform.Azure != nil:
		cc.Labels["region"] = cd.Spec.Platform.Azure.Region
	case cd.Spec.Platform.GCP != nil:
		cc.Labels["region"] = cd.Spec.Platform.GCP.Region
	}
	log.V(DEBUG).Info("Detected region: \"" + cc.Labels["region"] + "\"")

	return nil
}

func setClusterSetLabel(r *ClusterClaimsReconciler, cc *hivev1.ClusterClaim) error {
	var cp hivev1.ClusterPool
	if err := r.Client.Get(
		context.Background(),
		types.NamespacedName{Namespace: cc.Namespace,
			Name: cc.Spec.ClusterPoolName}, &cp); err != nil {

		if k8serrors.IsNotFound(err) {
			r.Log.V(WARN).Info("No Clusterpool found for " + cc.Name)
			return nil
		} else {
			return err
		}
	}
	//if clusterpool is no in a set, just return
	if cp.Labels == nil {
		return nil
	}
	if len(cp.Labels[ClusterSetLabel]) == 0 {
		return nil
	}

	// if the clusterclaim is already in a set, do nothing
	if cc.Labels == nil {
		cc.Labels = make(map[string]string)
	}
	if len(cc.Labels[ClusterSetLabel]) != 0 {
		return nil
	}

	// if the clusterclaim is not in a set, add the clusterpool's set label to clusterclaim
	cc.Labels[ClusterSetLabel] = cp.Labels[ClusterSetLabel]
	return nil
}
