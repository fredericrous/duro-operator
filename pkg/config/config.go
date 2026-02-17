package config

import (
	"fmt"
	"time"
)

// OperatorConfig holds the operator configuration
type OperatorConfig struct {
	// MetricsAddr is the address for the metrics endpoint
	MetricsAddr string

	// ProbeAddr is the address for health probes
	ProbeAddr string

	// EnableLeaderElection enables leader election
	EnableLeaderElection bool

	// LeaderElectionID is the ID for leader election
	LeaderElectionID string

	// MaxConcurrentReconciles is the maximum number of concurrent reconciles
	MaxConcurrentReconciles int

	// ReconcileTimeout is the timeout for reconcile operations
	ReconcileTimeout time.Duration

	// DuroNamespace is the namespace where duro is deployed
	DuroNamespace string

	// DuroConfigMapName is the name of the duro apps ConfigMap
	DuroConfigMapName string
}

// NewDefaultConfig creates a default configuration
func NewDefaultConfig() *OperatorConfig {
	return &OperatorConfig{
		MetricsAddr:             ":8080",
		ProbeAddr:               ":8081",
		EnableLeaderElection:    false,
		LeaderElectionID:        "duro-operator",
		MaxConcurrentReconciles: 3,
		ReconcileTimeout:        5 * time.Minute,
		DuroNamespace:           "duro",
		DuroConfigMapName:       "duro-apps",
	}
}

// Validate validates the configuration
func (c *OperatorConfig) Validate() error {
	if c.MaxConcurrentReconciles < 1 {
		return fmt.Errorf("maxConcurrentReconciles must be at least 1")
	}
	if c.ReconcileTimeout < time.Second {
		return fmt.Errorf("reconcileTimeout must be at least 1 second")
	}
	if c.DuroNamespace == "" {
		return fmt.Errorf("duroNamespace is required")
	}
	return nil
}
