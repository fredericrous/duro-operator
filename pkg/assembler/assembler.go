package assembler

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"

	"github.com/go-logr/logr"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
)

// Assembler handles DashboardApp configuration assembly
type Assembler struct {
	Log logr.Logger
}

// NewAssembler creates a new Assembler
func NewAssembler(log logr.Logger) *Assembler {
	return &Assembler{Log: log}
}

// AppEntry represents a single app in the output JSON
type AppEntry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Category string   `json:"category"`
	Icon     string   `json:"icon"`
	Groups   []string `json:"groups"`
	Priority int      `json:"priority"`
}

// categoryOrder defines the display order for categories
var categoryOrder = map[string]int{
	"media":        0,
	"ai":           1,
	"productivity": 2,
	"development":  3,
	"admin":        4,
}

// AssemblyResult contains the assembled JSON output
type AssemblyResult struct {
	Entries  []AppEntry
	AppsJSON string
}

// Assemble processes all DashboardApps and produces a JSON array
func (a *Assembler) Assemble(ctx context.Context, apps []dashboardv1alpha1.DashboardApp) (*AssemblyResult, error) {
	entries := make([]AppEntry, 0, len(apps))

	for _, app := range apps {
		priority := app.Spec.Priority
		if priority == 0 {
			priority = 100
		}

		entries = append(entries, AppEntry{
			ID:       app.Name,
			Name:     app.Spec.Name,
			URL:      app.Spec.URL,
			Category: app.Spec.Category,
			Icon:     app.Spec.Icon,
			Groups:   app.Spec.Groups,
			Priority: priority,
		})
	}

	// Sort by category order, then priority, then name
	slices.SortFunc(entries, func(a, b AppEntry) int {
		ca := categoryOrder[a.Category]
		cb := categoryOrder[b.Category]
		if c := cmp.Compare(ca, cb); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Priority, b.Priority); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	jsonBytes, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, err
	}

	return &AssemblyResult{
		Entries:  entries,
		AppsJSON: string(jsonBytes),
	}, nil
}
