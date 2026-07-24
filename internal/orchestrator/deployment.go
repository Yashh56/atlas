package orchestrator

import (
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

// ApprovalInfo tracks manual approval state.
type ApprovalInfo struct {
	Required   bool       `json:"required"`
	ApprovedBy *string    `json:"approved_by"`
	ApprovedAt *time.Time `json:"approved_at"`
}

// HealthCheckInfo tracks health-check state (not executed until Week 4+).
type HealthCheckInfo struct {
	URL            *string `json:"url"`
	ExpectedStatus int     `json:"expected_status"`
	Attempts       int     `json:"attempts"`
	MaxAttempts    int     `json:"max_attempts"`
}

// DeploymentState is created at session creation time. Owned by orchestrator.
type DeploymentState struct {
	Provider              string          `json:"provider"`
	Environment           string          `json:"environment"`
	DeploymentURL         *string         `json:"deployment_url"`
	PreviousDeploymentURL *string         `json:"previous_deployment_url"`
	RollbackAvailable     bool            `json:"rollback_available"`
	Approval              ApprovalInfo    `json:"approval"`
	HealthCheck           HealthCheckInfo `json:"health_check"`
	LastHealthyDeployment *deploy.Deployment `json:"last_healthy_deployment"`
}

const deploymentFile = "deployment.json"

// NewDeployment returns a DeploymentState with sane defaults for the given
// provider. The provider is the only value written this week.
func NewDeployment(provider string) *DeploymentState {
	return &DeploymentState{
		Provider:          provider,
		Environment:       "production",
		RollbackAvailable: false,
		Approval: ApprovalInfo{
			Required: true,
		},
		HealthCheck: HealthCheckInfo{
			ExpectedStatus: 200,
			Attempts:       0,
			MaxAttempts:    3,
		},
	}
}

// SaveDeployment atomically writes the deployment state to dir/deployment.json.
func SaveDeployment(dir string, d *DeploymentState) error {
	return state.SaveJSON(dir, deploymentFile, d)
}

// LoadDeployment reads deployment.json from dir.
func LoadDeployment(dir string) (*DeploymentState, error) {
	var d DeploymentState
	if err := state.LoadJSON(dir, deploymentFile, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
