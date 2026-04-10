package apiserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
)

// AppResponse represents a single application in the API response.
type AppResponse struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Category string   `json:"category"`
	Groups   []string `json:"groups"`
	Priority int      `json:"priority"`
}

// NewAppsHandler returns an http.Handler that lists DashboardApp CRs from the
// informer cache and returns them as a JSON array.
func NewAppsHandler(reader client.Reader, log logr.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		appList := &dashboardv1alpha1.DashboardAppList{}
		if err := reader.List(ctx, appList); err != nil {
			log.Error(err, "Failed to list DashboardApps from cache")
			http.Error(w, `{"error":"failed to list apps"}`, http.StatusInternalServerError)
			return
		}

		apps := make([]AppResponse, 0, len(appList.Items))
		for _, item := range appList.Items {
			apps = append(apps, AppResponse{
				ID:       item.Name, // metadata.name
				Name:     item.Spec.Name,
				URL:      item.Spec.URL,
				Category: item.Spec.Category,
				Groups:   item.Spec.Groups,
				Priority: item.Spec.Priority,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(apps); err != nil {
			log.Error(err, "Failed to encode apps response")
		}
	})
}
