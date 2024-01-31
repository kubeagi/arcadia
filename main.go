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

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	evaluationarcadiav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	chaincontrollers "github.com/kubeagi/arcadia/controllers/app-node/chain"
	promptcontrollers "github.com/kubeagi/arcadia/controllers/app-node/prompt"
	retrievertrollers "github.com/kubeagi/arcadia/controllers/app-node/retriever"
	basecontrollers "github.com/kubeagi/arcadia/controllers/base"
	evaluationcontrollers "github.com/kubeagi/arcadia/controllers/evaluation"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/utils"

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
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(apichain.AddToScheme(scheme))
	utilruntime.Must(apiprompt.AddToScheme(scheme))
	utilruntime.Must(apiretriever.AddToScheme(scheme))
	utilruntime.Must(evaluationarcadiav1alpha1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
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
	klog.InitFlags(nil)
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
	ctx := ctrl.SetupSignalHandler()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Validate if arcadia-config configMap exists before start all controllers
	clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		panic(err)
	}
	_, err = clientset.CoreV1().ConfigMaps(utils.GetCurrentNamespace()).Get(ctx, config.EnvConfigDefaultValue, metav1.GetOptions{})
	if err != nil {
		setupLog.Error(err, "failed to find required configMap", utils.GetCurrentNamespace(), config.EnvConfigDefaultValue)
		panic(err)
	}

	if err = (&basecontrollers.LLMReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LLM")
		os.Exit(1)
	}
	// Deprecated: will remove later, use promptcontrollers.PromptReconciler and construct a application
	if err = (&basecontrollers.PromptReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Prompt")
		os.Exit(1)
	}
	if err = (&basecontrollers.DatasourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Datasource")
		os.Exit(1)
	}
	if err = (&basecontrollers.EmbedderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Embedder")
		os.Exit(1)
	}
	if err = (&basecontrollers.DatasetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Dataset")
		os.Exit(1)
	}
	if err = (&basecontrollers.VersionedDatasetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VersionedDataset")
		os.Exit(1)
	}
	if err = (&basecontrollers.WorkerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Worker")
		os.Exit(1)
	}
	if err = (&basecontrollers.ModelReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Model")
		os.Exit(1)
	}
	if err = (&basecontrollers.KnowledgeBaseReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		HasHandledSuccessPath: make(map[string]bool),
		ReadyMap:              make(map[string]bool),
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KnowledgeBase")
		os.Exit(1)
	}
	if err = (&basecontrollers.VectorStoreReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorStore")
		os.Exit(1)
	}
	if err = (&basecontrollers.NamespaceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}
	if err = (&basecontrollers.ApplicationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	if err = (&chaincontrollers.LLMChainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LLMChain")
		os.Exit(1)
	}
	if err = (&chaincontrollers.RetrievalQAChainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RetrievalQAChain")
		os.Exit(1)
	}
	if err = (&retrievertrollers.KnowledgeBaseRetrieverReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KnowledgeBaseRetriever")
		os.Exit(1)
	}
	if err = (&promptcontrollers.PromptReconciler{
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
	if err = (&evaluationcontrollers.RAGReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RAG")
		os.Exit(1)
	}
	if err = (&chaincontrollers.APIChainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "APIChain")
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
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
