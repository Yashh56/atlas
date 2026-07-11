package orchestrator_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/orchestrator"
)

func TestCheckApproval_Auto(t *testing.T) {
	cfg := &config.Config{Approval: "auto"}
	dep := orchestrator.NewDeployment("vercel")
	
	var out bytes.Buffer
	approved := orchestrator.CheckApproval(cfg, dep, strings.NewReader(""), &out)
	
	if !approved {
		t.Errorf("expected auto approval to return true")
	}
	if out.String() != "" {
		t.Errorf("expected no output for auto approval, got %q", out.String())
	}
	if dep.Approval.ApprovedBy == nil || *dep.Approval.ApprovedBy != "auto" {
		t.Errorf("expected ApprovedBy to be 'auto'")
	}
}

func TestCheckApproval_ManualYes(t *testing.T) {
	cfg := &config.Config{Approval: "manual"}
	dep := orchestrator.NewDeployment("vercel")
	
	var out bytes.Buffer
	approved := orchestrator.CheckApproval(cfg, dep, strings.NewReader("y\n"), &out)
	
	if !approved {
		t.Errorf("expected manual 'y' to return true")
	}
	if !strings.Contains(out.String(), "[y/N]") {
		t.Errorf("expected prompt in output, got %q", out.String())
	}
	if dep.Approval.ApprovedBy == nil || *dep.Approval.ApprovedBy != "cli-user" {
		t.Errorf("expected ApprovedBy to be 'cli-user'")
	}
}

func TestCheckApproval_ManualNo(t *testing.T) {
	cfg := &config.Config{Approval: "manual"}
	dep := orchestrator.NewDeployment("vercel")
	
	var out bytes.Buffer
	approved := orchestrator.CheckApproval(cfg, dep, strings.NewReader("n\n"), &out)
	
	if approved {
		t.Errorf("expected manual 'n' to return false")
	}
}

func TestCheckApproval_ManualGarbage(t *testing.T) {
	cfg := &config.Config{Approval: "manual"}
	dep := orchestrator.NewDeployment("vercel")
	
	var out bytes.Buffer
	approved := orchestrator.CheckApproval(cfg, dep, strings.NewReader("garbage\n"), &out)
	
	if approved {
		t.Errorf("expected garbage input to return false")
	}
}

func TestRecordDeployment(t *testing.T) {
	dep := orchestrator.NewDeployment("vercel")
	
	orchestrator.RecordDeployment(dep, "https://v1.com")
	if dep.DeploymentURL == nil || *dep.DeploymentURL != "https://v1.com" {
		t.Errorf("expected DeploymentURL to be v1")
	}
	if dep.PreviousDeploymentURL != nil {
		t.Errorf("expected PreviousDeploymentURL to be nil")
	}
	
	orchestrator.RecordDeployment(dep, "https://v2.com")
	if dep.DeploymentURL == nil || *dep.DeploymentURL != "https://v2.com" {
		t.Errorf("expected DeploymentURL to be v2")
	}
	if dep.PreviousDeploymentURL == nil || *dep.PreviousDeploymentURL != "https://v1.com" {
		t.Errorf("expected PreviousDeploymentURL to be v1")
	}
}
