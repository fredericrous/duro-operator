package assembler

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
)

func TestAssembler_Assemble(t *testing.T) {
	log := zap.New(zap.UseDevMode(true))
	a := NewAssembler(log)

	apps := []dashboardv1alpha1.DashboardApp{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plex", Namespace: "plex"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Plex",
				URL:      "https://plex.example.com",
				Category: "media",
				Icon:     "<svg>plex</svg>",
				Groups:   []string{"friends", "family", "lldap_admin"},
				Priority: 10,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "gitea", Namespace: "gitea"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Gitea",
				URL:      "https://gitea.example.com",
				Category: "development",
				Icon:     "<svg>gitea</svg>",
				Groups:   []string{"lldap_admin"},
				Priority: 20,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "openwebui", Namespace: "openwebui"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "OpenWebUI",
				URL:      "https://ai.example.com",
				Category: "ai",
				Icon:     "<svg>ai</svg>",
				Groups:   []string{"friends", "family", "lldap_admin"},
				Priority: 10,
			},
		},
	}

	result, err := a.Assemble(context.Background(), apps)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(result.Entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(result.Entries))
	}

	// Verify sort order: media (0) → ai (1) → development (3)
	if result.Entries[0].Category != "media" {
		t.Errorf("Expected first entry category 'media', got '%s'", result.Entries[0].Category)
	}
	if result.Entries[1].Category != "ai" {
		t.Errorf("Expected second entry category 'ai', got '%s'", result.Entries[1].Category)
	}
	if result.Entries[2].Category != "development" {
		t.Errorf("Expected third entry category 'development', got '%s'", result.Entries[2].Category)
	}

	// Verify JSON output parses correctly
	var entries []AppEntry
	if err := json.Unmarshal([]byte(result.AppsJSON), &entries); err != nil {
		t.Fatalf("Failed to unmarshal AppsJSON: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries in JSON, got %d", len(entries))
	}
}

func TestAssembler_SortWithinCategory(t *testing.T) {
	log := zap.New(zap.UseDevMode(true))
	a := NewAssembler(log)

	apps := []dashboardv1alpha1.DashboardApp{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sonarr", Namespace: "sonarr"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Sonarr",
				URL:      "https://sonarr.example.com",
				Category: "media",
				Icon:     "<svg/>",
				Groups:   []string{"lldap_admin"},
				Priority: 50,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plex", Namespace: "plex"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Plex",
				URL:      "https://plex.example.com",
				Category: "media",
				Icon:     "<svg/>",
				Groups:   []string{"friends"},
				Priority: 10,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "seerr", Namespace: "seerr"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "Seerr",
				URL:      "https://seerr.example.com",
				Category: "media",
				Icon:     "<svg/>",
				Groups:   []string{"lldap_admin"},
				Priority: 40,
			},
		},
	}

	result, err := a.Assemble(context.Background(), apps)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	// Verify sort by priority within media category
	expected := []string{"Plex", "Seerr", "Sonarr"}
	for i, name := range expected {
		if result.Entries[i].Name != name {
			t.Errorf("Expected entry %d to be '%s', got '%s'", i, name, result.Entries[i].Name)
		}
	}
}

func TestAssembler_DefaultPriority(t *testing.T) {
	log := zap.New(zap.UseDevMode(true))
	a := NewAssembler(log)

	apps := []dashboardv1alpha1.DashboardApp{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "default"},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     "App1",
				URL:      "https://app1.example.com",
				Category: "admin",
				Icon:     "<svg/>",
				Groups:   []string{"lldap_admin"},
				// Priority omitted — should default to 100
			},
		},
	}

	result, err := a.Assemble(context.Background(), apps)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.Entries[0].Priority != 100 {
		t.Errorf("Expected default priority 100, got %d", result.Entries[0].Priority)
	}
}

func TestAssembler_EmptyInput(t *testing.T) {
	log := zap.New(zap.UseDevMode(true))
	a := NewAssembler(log)

	result, err := a.Assemble(context.Background(), nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(result.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(result.Entries))
	}

	// Empty array should still be valid JSON
	var entries []AppEntry
	if err := json.Unmarshal([]byte(result.AppsJSON), &entries); err != nil {
		t.Fatalf("Failed to unmarshal empty AppsJSON: %v", err)
	}
}
