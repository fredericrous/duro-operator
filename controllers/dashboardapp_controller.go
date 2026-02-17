package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
	"github.com/fredericrous/duro-operator/pkg/assembler"
	"github.com/fredericrous/duro-operator/pkg/config"
	operrors "github.com/fredericrous/duro-operator/pkg/errors"
)

// DashboardAppReconciler reconciles DashboardApp objects
type DashboardAppReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	Config    *config.OperatorConfig
	Assembler *assembler.Assembler
}

// SetupWithManager sets up the controller with the Manager
func (r *DashboardAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Config == nil {
		r.Config = config.NewDefaultConfig()
	}

	r.Assembler = assembler.NewAssembler(r.Log.WithName("assembler"))

	opts := controller.Options{
		MaxConcurrentReconciles: r.Config.MaxConcurrentReconciles,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dashboardv1alpha1.DashboardApp{}).
		WithOptions(opts).
		Complete(r)
}

// Reconcile handles the reconciliation loop
// +kubebuilder:rbac:groups=dashboard.homelab.io,resources=dashboardapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dashboard.homelab.io,resources=dashboardapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

func (r *DashboardAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("dashboardapp", req.NamespacedName, "trace_id", generateTraceID())

	ctx, cancel := context.WithTimeout(ctx, r.Config.ReconcileTimeout)
	defer cancel()

	ctx = logr.NewContext(ctx, log)

	log.V(1).Info("Starting reconciliation")

	// Fetch all DashboardApps cluster-wide
	appList := &dashboardv1alpha1.DashboardAppList{}
	if err := r.List(ctx, appList); err != nil {
		return ctrl.Result{}, operrors.NewTransientError("failed to list DashboardApps", err)
	}

	if len(appList.Items) == 0 {
		log.Info("No DashboardApp resources found, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Assemble the apps JSON
	result, err := r.Assembler.Assemble(ctx, appList.Items)
	if err != nil {
		r.Recorder.Event(&appList.Items[0], corev1.EventTypeWarning, "AssemblyFailed", err.Error())
		if operrors.ShouldRetry(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Update the duro apps ConfigMap
	if err := r.updateAppsConfig(ctx, result); err != nil {
		r.Recorder.Eventf(&appList.Items[0], corev1.EventTypeWarning, "ConfigUpdateFailed", "Failed to update duro apps config: %v", err)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status for all DashboardApps
	now := metav1.Now()
	var statusUpdateErrors []error
	for i := range appList.Items {
		app := &appList.Items[i]
		app.Status.Ready = true
		app.Status.LastSyncedAt = &now
		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "Failed to update DashboardApp status", "app", app.Name)
			statusUpdateErrors = append(statusUpdateErrors, err)
		}
	}

	if len(statusUpdateErrors) > 0 {
		log.Info("Some status updates failed, requeueing", "failedCount", len(statusUpdateErrors))
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Reconciliation completed successfully", "appCount", len(appList.Items))

	r.Recorder.Event(&appList.Items[0], corev1.EventTypeNormal, "Synced",
		fmt.Sprintf("Successfully assembled %d dashboard apps", len(appList.Items)))

	return ctrl.Result{}, nil
}

// updateAppsConfig updates the duro apps ConfigMap
func (r *DashboardAppReconciler) updateAppsConfig(ctx context.Context, result *assembler.AssemblyResult) error {
	log := logr.FromContextOrDiscard(ctx)

	configHash := computeHash(result.AppsJSON)

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: r.Config.DuroConfigMapName, Namespace: r.Config.DuroNamespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.Config.DuroConfigMapName,
					Namespace: r.Config.DuroNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "duro-operator",
					},
					Annotations: map[string]string{
						"dashboard.homelab.io/config-hash": configHash,
					},
				},
				Data: map[string]string{
					"apps.json": result.AppsJSON,
				},
			}
			log.Info("Creating duro apps ConfigMap", "name", r.Config.DuroConfigMapName)
			return r.Create(ctx, cm)
		}
		return err
	}

	existingHash := ""
	if existing.Annotations != nil {
		existingHash = existing.Annotations["dashboard.homelab.io/config-hash"]
	}

	if existingHash == configHash {
		log.V(1).Info("Duro apps ConfigMap unchanged (hash match), skipping update")
		return nil
	}

	existing.Data = map[string]string{
		"apps.json": result.AppsJSON,
	}
	if existing.Labels == nil {
		existing.Labels = make(map[string]string)
	}
	existing.Labels["app.kubernetes.io/managed-by"] = "duro-operator"
	if existing.Annotations == nil {
		existing.Annotations = make(map[string]string)
	}
	existing.Annotations["dashboard.homelab.io/config-hash"] = configHash

	log.Info("Updating duro apps ConfigMap", "name", r.Config.DuroConfigMapName, "hash", configHash)
	return r.Update(ctx, existing)
}

func computeHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
