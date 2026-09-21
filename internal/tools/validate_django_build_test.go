package tools_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestValidateDjangoBuild(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("python not found, skipping TestValidateDjangoBuild")
		}
	}
	
	wsRoot, _ := filepath.Abs("../../fixtures/django-ready")
	sessDir := t.TempDir()
	sess := session.New(wsRoot)
	
	venvPath := filepath.Join(sessDir, "ephemeral-venv")
	
	val := tools.ValidateDjangoBuild{
		WorkspaceRoot: wsRoot,
		VenvPath:      venvPath,
	}

	res, err := val.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected validation to succeed, got error: %s", res.Error)
	}

	// Verify venv was deleted
	if _, err := os.Stat(venvPath); !os.IsNotExist(err) {
		t.Errorf("ephemeral venv was not deleted: %v", err)
	}
}
