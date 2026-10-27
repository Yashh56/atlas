package orchestrator_test

import (
	"testing"

	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
)

// TestContextFiles_Isolation verifies that mutating one of the four state files
// does not affect the others — proving per-file ownership, not just round-trip.
func TestContextFiles_Isolation(t *testing.T) {
	dir := t.TempDir()

	// Create and save session.
	sess := session.New("/tmp/proj")
	if err := sess.Save(dir); err != nil {
		t.Fatalf("session.Save: %v", err)
	}
	sessDir := session.SessionDir(dir, sess.ID)

	// Create and save the other three files.
	planner := orchestrator.NewPlanner("deploy")
	if err := orchestrator.SavePlanner(sessDir, planner); err != nil {
		t.Fatalf("SavePlanner: %v", err)
	}

	proj := &orchestrator.ProjectState{}
	if err := orchestrator.SaveProject(sessDir, proj); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	dep := orchestrator.NewDeployment("vercel")
	if err := orchestrator.SaveDeployment(sessDir, dep); err != nil {
		t.Fatalf("SaveDeployment: %v", err)
	}

	// Mutate only planner.
	planner.CurrentStep = "analyze_project"
	planner.Completed = append(planner.Completed, "init")
	if err := orchestrator.SavePlanner(sessDir, planner); err != nil {
		t.Fatalf("SavePlanner (mutate): %v", err)
	}

	// Reload session — must be unchanged.
	loadedSess, err := session.Load(dir, sess.ID)
	if err != nil {
		t.Fatalf("session.Load: %v", err)
	}
	if loadedSess.Status != sess.Status {
		t.Errorf("session.Status changed after planner mutation: got %q, want %q", loadedSess.Status, sess.Status)
	}

	// Reload project — must be unchanged.
	loadedProj, err := orchestrator.LoadProject(sessDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if loadedProj.Framework != nil {
		t.Errorf("project.Framework changed after planner mutation: got %v, want nil", loadedProj.Framework)
	}

	// Reload deployment — must be unchanged.
	loadedDep, err := orchestrator.LoadDeployment(sessDir)
	if err != nil {
		t.Fatalf("LoadDeployment: %v", err)
	}
	if loadedDep.Provider != "vercel" {
		t.Errorf("deployment.Provider changed after planner mutation: got %q, want %q", loadedDep.Provider, "vercel")
	}

	// Mutate only deployment provider (simulating a future write).
	dep.RollbackAvailable = true
	if err := orchestrator.SaveDeployment(sessDir, dep); err != nil {
		t.Fatalf("SaveDeployment (mutate): %v", err)
	}

	// Reload planner — must still have the earlier mutation, not reset.
	loadedPlanner, err := orchestrator.LoadPlanner(sessDir)
	if err != nil {
		t.Fatalf("LoadPlanner: %v", err)
	}
	if loadedPlanner.CurrentStep != "analyze_project" {
		t.Errorf("planner.CurrentStep changed after deployment mutation: got %q, want %q", loadedPlanner.CurrentStep, "analyze_project")
	}
}

func TestPlannerState_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	p := orchestrator.NewPlanner("deploy my service")
	if err := orchestrator.SavePlanner(dir, p); err != nil {
		t.Fatalf("SavePlanner: %v", err)
	}

	loaded, err := orchestrator.LoadPlanner(dir)
	if err != nil {
		t.Fatalf("LoadPlanner: %v", err)
	}

	if loaded.Goal != p.Goal {
		t.Errorf("Goal mismatch: got %q, want %q", loaded.Goal, p.Goal)
	}
	if len(loaded.Pending) != len(p.Pending) {
		t.Errorf("Pending length mismatch: got %d, want %d", len(loaded.Pending), len(p.Pending))
	}
}

func TestDeploymentState_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	d := orchestrator.NewDeployment("render")
	if err := orchestrator.SaveDeployment(dir, d); err != nil {
		t.Fatalf("SaveDeployment: %v", err)
	}

	loaded, err := orchestrator.LoadDeployment(dir)
	if err != nil {
		t.Fatalf("LoadDeployment: %v", err)
	}

	if loaded.Provider != "render" {
		t.Errorf("Provider mismatch: got %q, want %q", loaded.Provider, "render")
	}
	if loaded.HealthCheck.MaxAttempts != 3 {
		t.Errorf("HealthCheck.MaxAttempts mismatch: got %d, want 3", loaded.HealthCheck.MaxAttempts)
	}
}

func TestNormalizeErrorSignature(t *testing.T) {
	err1 := "main.go:5:10: undefined: Foo\nexit status 1"
	err2 := "  main.go:5:10: undefined: Foo \n\n exit status 1\n"
	err3 := "main.go:6:12: syntax error\nexit status 1"

	sig1 := orchestrator.NormalizeErrorSignature(err1)
	sig2 := orchestrator.NormalizeErrorSignature(err2)
	sig3 := orchestrator.NormalizeErrorSignature(err3)

	if sig1 != sig2 {
		t.Errorf("expected trimmed error signatures to match: %q != %q", sig1, sig2)
	}
	if sig1 == sig3 {
		t.Errorf("expected different errors to have different signatures: %q == %q", sig1, sig3)
	}
}
