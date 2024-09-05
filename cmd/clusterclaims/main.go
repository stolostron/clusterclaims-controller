// Copyright Contributors to the Open Cluster Management project.

package main

import (
	"flag"
	"os"
	"time"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	controller "github.com/stolostron/clusterclaims-controller/controllers/clusterclaims"
	managedclustercontroller "github.com/stolostron/clusterclaims-controller/controllers/managedcluster"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	mcv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = hivev1.AddToScheme(scheme)
	_ = mcv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var leaderElectionLeaseDuration time.Duration
	var leaderElectionRenewDeadline time.Duration
	var leaderElectionRetryPeriod time.Duration
	flag.StringVar(&metricsAddr, "metrics-addr", ":9443", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(
		&leaderElectionLeaseDuration,
		"leader-election-lease-duration",
		137*time.Second,
		"The duration that non-leader candidates will wait after observing a leadership "+
			"renewal until attempting to acquire leadership of a led but unrenewed leader "+
			"slot. This is effectively the maximum duration that a leader can be stopped "+
			"before it is replaced by another candidate. This is only applicable if leader "+
			"election is enabled.",
	)
	flag.DurationVar(
		&leaderElectionRenewDeadline,
		"leader-election-renew-deadline",
		107*time.Second,
		"The interval between attempts by the acting master to renew a leadership slot "+
			"before it stops leading. This must be less than or equal to the lease duration. "+
			"This is only applicable if leader election is enabled.",
	)
	flag.DurationVar(
		&leaderElectionRetryPeriod,
		"leader-election-retry-period",
		26*time.Second,
		"The duration the clients should wait between attempting acquisition and renewal "+
			"of a leadership. This is only applicable if leader election is enabled.",
	)
	flag.Parse()

	// To run in debug change zapcore.InfoLevel to zapcore.DebugLevel
	ctrl.SetLogger(zap.New(zap.Level(zapcore.InfoLevel)))

	setupLog.Info("Leader election settings", "enableLeaderElection", enableLeaderElection,
		"leaseDuration", leaderElectionLeaseDuration,
		"renewDeadline", leaderElectionRenewDeadline,
		"retryPeriod", leaderElectionRetryPeriod)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "clusterclaims-controller.open-cluster-management.io",
		LeaseDuration:    &leaderElectionLeaseDuration,
		RenewDeadline:    &leaderElectionRenewDeadline,
		RetryPeriod:      &leaderElectionRetryPeriod,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.ClusterClaimsReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controller").WithName("ClusterClaimsReconciler"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create claim controller", "controller")
		os.Exit(1)
	}

	if err = (&managedclustercontroller.ManagedClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controller").WithName("ManagedClusterReconciler"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create managed cluster controller", "controller")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
