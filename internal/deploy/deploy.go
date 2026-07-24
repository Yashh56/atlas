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
	DeployedAt time.Time
}

// Provider is the interface for all deployment platforms.
type Provider interface {
	Name() string
	Deploy(ctx context.Context, in DeployInput) (*Deployment, error)
}
