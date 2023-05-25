/*
Copyright 2023 The Primaza Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	EnvWatchNamespace              = "WATCH_NAMESPACE"
	EnvAppAgentImage               = "AGENT_APP_IMAGE"
	EnvSvcAgentImage               = "AGENT_SVC_IMAGE"
	EnvHealthCheckInterval         = "HEALTH_CHECK_INTERVAL"
	DefaultHealthCheckInterval int = 600
	MinimumHealtCheckInterval  int = 10
	EnvAppAgentManifest            = "AGENT_APP_MANIFEST"
	EnvSvcAgentManifest            = "AGENT_SVC_MANIFEST"
	EnvAppAgentConfigManifest      = "AGENT_APP_CONFIG_MANIFEST"
	EnvSvcAgentConfigManifest      = "AGENT_SVC_CONFIG_MANIFEST"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(primazaiov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg, err := getConfig(setupLog)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	setupLog.Info("got configuration", "configuration", cfg)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		Namespace:              cfg.WatchNamespace,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "859ca7e5.primaza.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	cerConfig := controllers.ClusterEnvironmentReconcilerConfig{
		ControlPlaneNamespace:  cfg.WatchNamespace,
		AppAgentImage:          cfg.AppImage,
		SvcAgentImage:          cfg.SvcImage,
		AppAgentManifest:       cfg.AppAgentManifest,
		AppAgentConfigManifest: cfg.AppAgentConfigManifest,
		SvcAgentManifest:       cfg.SvcAgentManifest,
		SvcAgentConfigManifest: cfg.SvcAgentConfigManifest,
	}
	cer := controllers.NewClusterEnvironmentReconciler(mgr, cerConfig)
	if err = cer.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterEnvironment")
		os.Exit(1)
	}

	serviceClaimController := controllers.NewServiceClaimReconciler(mgr)
	if err := serviceClaimController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceClaim")
		os.Exit(1)
	}
	if err = (&controllers.ServiceClassReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceClass")
		os.Exit(1)
	}
	if err = (&primazaiov1alpha1.ServiceClass{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ServiceClass")
		os.Exit(1)
	}
	if err = (&controllers.RegisteredServiceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RegisteredService")
		os.Exit(1)
	}

	if err = (&primazaiov1alpha1.ServiceClaim{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ServiceClaim")
		os.Exit(1)
	}

	if err = (&controllers.ServiceCatalogReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceCatalog")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.Add(&clusterEnvironmentHealthcheck{
		cer: cer,
		ns:  cfg.WatchNamespace,
		hci: cfg.HealthCheckInterval,
	}); err != nil {
		setupLog.Error(err, "unable to set up ClusterEnvironments healthchecks")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getHealthCheckIntervalFromEnv(log logr.Logger) int {
	ehci := os.Getenv(EnvHealthCheckInterval)
	hci, err := strconv.Atoi(ehci)
	if err != nil {
		log.Info(
			"HealthCheckInterval not set or not a integer value: using default value",
			"interval", hci,
			"error", err.Error(),
			"default", DefaultHealthCheckInterval)
		return DefaultHealthCheckInterval
	}
	if hci < MinimumHealtCheckInterval {
		log.Info(
			"provided HealthCheckInterval lower than minimum: using minimum value",
			"interval", hci,
			"minimum", MinimumHealtCheckInterval,
			"default", DefaultHealthCheckInterval)
		return MinimumHealtCheckInterval
	}

	return hci
}

type clusterEnvironmentHealthcheck struct {
	cer *controllers.ClusterEnvironmentReconciler
	ns  string
	hci int
}

func (h *clusterEnvironmentHealthcheck) Start(ctx context.Context) error {
	go h.cer.MonitorHealth(ctx, h.ns, h.hci)
	return nil
}

type config struct {
	WatchNamespace         string
	AppImage               string
	SvcImage               string
	HealthCheckInterval    int
	AppAgentManifest       string
	SvcAgentManifest       string
	AppAgentConfigManifest string
	SvcAgentConfigManifest string
}

func getConfig(log logr.Logger) (*config, error) {
	ns, err := getRequiredEnv(EnvWatchNamespace)
	if err != nil {
		return nil, err
	}

	ai, err := getRequiredEnv(EnvAppAgentImage)
	if err != nil {
		return nil, err
	}

	si, err := getRequiredEnv(EnvSvcAgentImage)
	if err != nil {
		return nil, err
	}

	hci := getHealthCheckIntervalFromEnv(log)

	as, err := getRequiredEnv(EnvAppAgentManifest)
	if err != nil {
		return nil, err
	}

	ss, err := getRequiredEnv(EnvSvcAgentManifest)
	if err != nil {
		return nil, err
	}

	acm, err := getRequiredEnv(EnvAppAgentConfigManifest)
	if err != nil {
		return nil, err
	}

	scm, err := getRequiredEnv(EnvSvcAgentConfigManifest)
	if err != nil {
		return nil, err
	}

	return &config{
		WatchNamespace:         ns,
		AppImage:               ai,
		SvcImage:               si,
		HealthCheckInterval:    hci,
		AppAgentManifest:       as,
		SvcAgentManifest:       ss,
		AppAgentConfigManifest: acm,
		SvcAgentConfigManifest: scm,
	}, nil
}

func getRequiredEnv(env string) (string, error) {
	ns := os.Getenv(env)
	if ns == "" {
		return "", fmt.Errorf("environment variable %s not found", env)
	}

	return ns, nil
}
