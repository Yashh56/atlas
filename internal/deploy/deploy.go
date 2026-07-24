package deploy

import (
	"context"
	"time"
)

type DeployInput struct {
	WorkspaceRoot string
	SessionDir    string // directory of the current session state
	Environment   string // "production" for now
	OutputDir     string // manual output dir override from CLI
	Token         string // resolved provider token, if any
}

// Deployment holds the result of a successful deployment.
type Deployment struct {
	URL        string
	Provider   string
	ProviderRef string // opaque reference used for rollback (e.g., deploy ID, commit SHA, or URL)
	DeployedAt time.Time
}

// Provider is the interface for all deployment platforms.
type Provider interface {
	Name() string
	Deploy(ctx context.Context, in DeployInput) (*Deployment, error)
	HealthCheck(ctx context.Context, d *Deployment) error
	Rollback(ctx context.Context, to *Deployment, in DeployInput) error
}
