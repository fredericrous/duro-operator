package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("register corev1: %v", err)
	}
	if err := dashboardv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("register dashboard v1alpha1: %v", err)
	}
	return s
}

func TestNewAppsHandler_ListsAppsAsJSON(t *testing.T) {
	s := newScheme(t)
	apps := []client.Object{
		&dashboardv1alpha1.DashboardApp{
			ObjectMeta: metav1.ObjectMeta{Name: "plex", Namespace: "plex"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Plex",
				URL:      "https://plex.example",
				Category: "media",
				Icon:     "<svg/>",
				Groups:   []string{"users"},
				Priority: 10,
			},
		},
		&dashboardv1alpha1.DashboardApp{
			ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "monitoring"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Grafana",
				URL:      "https://grafana.example",
				Category: "ops",
				Icon:     "<svg/>",
				Groups:   []string{"admins"},
				Priority: 5,
			},
		},
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(apps...).Build()

	h := NewAppsHandler(c, logr.Discard())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/apps", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var got []AppResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d apps, want 2; body=%s", len(got), rr.Body.String())
	}
	// ID must be metadata.name, not spec.name.
	byID := map[string]AppResponse{}
	for _, a := range got {
		byID[a.ID] = a
	}
	if byID["plex"].Name != "Plex" || byID["plex"].Priority != 10 {
		t.Errorf("plex app mismatch: %+v", byID["plex"])
	}
	if byID["grafana"].URL != "https://grafana.example" {
		t.Errorf("grafana URL mismatch: %+v", byID["grafana"])
	}
}

func TestNewAppsHandler_MethodNotAllowed(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(newScheme(t)).Build()
	h := NewAppsHandler(c, logr.Discard())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/apps", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

// erroringReader always fails on List, simulating a broken cache.
type erroringReader struct{ client.Reader }

func (erroringReader) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return errors.New("not implemented")
}
func (erroringReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return errors.New("boom")
}

func TestNewAppsHandler_ListError(t *testing.T) {
	h := NewAppsHandler(erroringReader{}, logr.Discard())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/apps", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}
