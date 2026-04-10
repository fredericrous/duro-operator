package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
	"github.com/fredericrous/duro-operator/controllers"
	"github.com/fredericrous/duro-operator/pkg/apiserver"
	"github.com/fredericrous/duro-operator/pkg/config"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dashboardv1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          = flag.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to")
		probeAddr            = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to")
		enableLeaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager")
		leaderElectionID     = flag.String("leader-election-id", "duro-operator", "Leader election ID")

		maxConcurrentReconciles = flag.Int("max-concurrent-reconciles", 3, "Maximum number of concurrent reconciles")
		reconcileTimeout        = flag.Duration("reconcile-timeout", 5*time.Minute, "Timeout for each reconcile operation")

		apiAddr           = flag.String("api-bind-address", ":9090", "The address the REST API binds to")

		duroNamespace     = flag.String("duro-namespace", "duro", "Namespace where duro is deployed")
		duroConfigMapName = flag.String("duro-configmap", "duro-apps", "Name of the duro apps ConfigMap")

		logLevel   = flag.String("zap-log-level", "info", "Zap log level (debug, info, warn, error)")
		logDevel   = flag.Bool("zap-devel", false, "Enable development mode logging")
		logEncoder = flag.String("zap-encoder", "json", "Zap log encoding (json or console)")
	)

	flag.Parse()

	opts := zap.Options{
		Development: *logDevel,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}

	switch *logLevel {
	case "debug":
		opts.Level = zapcore.DebugLevel
	case "info":
		opts.Level = zapcore.InfoLevel
	case "warn":
		opts.Level = zapcore.WarnLevel
	case "error":
		opts.Level = zapcore.ErrorLevel
	default:
		opts.Level = zapcore.InfoLevel
	}

	if *logEncoder == "console" {
		opts.Encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg := &config.OperatorConfig{
		MetricsAddr:             *metricsAddr,
		ProbeAddr:               *probeAddr,
		ApiAddr:                 *apiAddr,
		EnableLeaderElection:    *enableLeaderElection,
		LeaderElectionID:        *leaderElectionID,
		MaxConcurrentReconciles: *maxConcurrentReconciles,
		ReconcileTimeout:        *reconcileTimeout,
		DuroNamespace:           *duroNamespace,
		DuroConfigMapName:       *duroConfigMapName,
	}

	if err := cfg.Validate(); err != nil {
		setupLog.Error(err, "Invalid configuration")
		os.Exit(1)
	}

	setupLog.Info("Starting duro-operator",
		"duroNamespace", cfg.DuroNamespace,
		"metricsAddr", cfg.MetricsAddr,
		"probeAddr", cfg.ProbeAddr,
		"enableLeaderElection", cfg.EnableLeaderElection,
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: cfg.MetricsAddr},
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.EnableLeaderElection,
		LeaderElectionID:       cfg.LeaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "Failed to create manager")
		os.Exit(1)
	}

	recorder := mgr.GetEventRecorderFor("duro-operator")

	reconciler := &controllers.DashboardAppReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("DashboardApp"),
		Scheme:   mgr.GetScheme(),
		Recorder: recorder,
		Config:   cfg,
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to setup controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLog.Error(err, "Failed to add health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLog.Error(err, "Failed to add readiness check")
		os.Exit(1)
	}

	// Start REST API server as a managed runnable
	if cfg.ApiAddr != "" && cfg.ApiAddr != "0" {
		apiMux := http.NewServeMux()
		apiMux.Handle("/api/v1/apps", apiserver.NewAppsHandler(mgr.GetClient(), ctrl.Log.WithName("apiserver")))
		apiMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		apiSrv := &http.Server{Addr: cfg.ApiAddr, Handler: apiMux}

		if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			go func() {
				<-ctx.Done()
				apiSrv.Shutdown(context.Background())
			}()
			setupLog.Info("Starting API server", "addr", cfg.ApiAddr)
			if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		})); err != nil {
			setupLog.Error(err, "Failed to add API server runnable")
			os.Exit(1)
		}
	}

	setupLog.Info("Starting manager")
	ctx := ctrl.SetupSignalHandler()
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "Failed to run manager")
		os.Exit(1)
	}
}
