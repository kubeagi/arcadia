/*
Copyright 2023 KubeAGI.

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
	"flag"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/controllers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(arcadiav1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		configFile      string
		enableProfiling bool
		probeAddr       string
	)
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.BoolVar(&enableProfiling, "profiling", true,
		"Enable profiling via web interface host:port/debug/pprof/")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	options := ctrl.Options{Scheme: scheme, HealthProbeBindAddress: probeAddr}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	var enableWebhooks bool
	// 1. Environment variable has the highest priority
	v, ok := os.LookupEnv("ENABLE_WEBHOOKS")
	if !ok {
		// 2. options.CertDir can be configured through the config file, priority 2
		if options.CertDir != "" {
			enableWebhooks = true
		} else {
			// 3. The default directory has a value of priority 3
			defaultPath := filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
			_, err := os.Stat(defaultPath)
			if err == nil {
				enableWebhooks = true
			}
			if err != nil {
				if os.IsNotExist(err) {
					enableWebhooks = false
				}
			}
		}
	} else {
		// 4. If the environment variable is configured, but there is a configuration error, exit directly.
		enableWebhooks, err = strconv.ParseBool(v)
		if err != nil {
			setupLog.Error(err, "unable to parse ENABLE_WEBHOOKS")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.LaboratoryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Laboratory")
		os.Exit(1)
	}
	if err = (&controllers.LLMReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LLM")
		os.Exit(1)
	}
	if err = (&controllers.PromptReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Prompt")
		os.Exit(1)
	}
	if enableWebhooks {
		if err = (&arcadiav1alpha1.Prompt{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Prompt")
			os.Exit(1)
		}
	}

	if err = (&controllers.DatasourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Datasource")
		os.Exit(1)
	}
	if err = (&controllers.EmbedderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Embedder")
		os.Exit(1)
	}
	if err = (&controllers.DatasetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Dataset")
		os.Exit(1)
	}
	if err = (&controllers.VersionedDatasetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VersionedDataset")
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

	if enableProfiling {
		_ = mgr.AddMetricsExtraHandler("/debug/pprof/", http.HandlerFunc(pprof.Index))
		_ = mgr.AddMetricsExtraHandler("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		_ = mgr.AddMetricsExtraHandler("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		_ = mgr.AddMetricsExtraHandler("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		_ = mgr.AddMetricsExtraHandler("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// TODO: Defines the configuration for the Arcadia controller and implement the config parser
// Config defines the configuration for the Arcadia controller
type Config struct {
	// SystemDatasource specifies the built-in datasource for Arcadia to host data files and model files
	SystemDatasource arcadiav1alpha1.TypedObjectReference
}
