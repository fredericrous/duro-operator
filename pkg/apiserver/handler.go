package apiserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

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
//
// The response carries a strong ETag (a hash of the exact body) and honours
// If-None-Match with 304, so a consumer can poll it cheaply: duro's worker
// asks every minute and only runs its app sync when the list really changed.
// The list is sorted by ID first; the informer cache promises no order, and an
// order-dependent hash would change on every call and defeat the point.
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
		sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })

		body, err := json.Marshal(apps)
		if err != nil {
			log.Error(err, "Failed to encode apps response")
			http.Error(w, `{"error":"failed to encode apps"}`, http.StatusInternalServerError)
			return
		}
		sum := sha256.Sum256(body)
		etag := `"` + hex.EncodeToString(sum[:16]) + `"`

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		if etagMatches(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(append(body, '\n')); err != nil {
			log.Error(err, "Failed to write apps response")
		}
	})
}

// etagMatches reports whether an If-None-Match header names etag (RFC 9110:
// a comma-separated list, or "*"; weak validators compare equal here, since
// the body is byte-identical whenever the hash is).
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
