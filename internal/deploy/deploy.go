package deploy

import (
	"context"
	"time"
)

// DeployInput holds parameters for a deployment.
type DeployInput struct {
	WorkspaceRoot string
	SessionDir    string // directory of the current session state
	Environment   string // "production" for now
}

// Deployment holds the result of a successful deployment.
type Deployment struct {
	URL        string
	Provider   string
	DeployedAt time.Time
}

// Provider is the interface for all deployment platforms.
type Provider interface {
	Name() string
	Deploy(ctx context.Context, in DeployInput) (*Deployment, error)
}
