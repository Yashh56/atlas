package doctor_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Yashh56/atlas/internal/doctor"
)

func TestRunDiagnostics(t *testing.T) {
	report, err := doctor.RunDiagnostics(context.Background(), ".")
	if err != nil {
		t.Fatalf("unexpected error running diagnostics: %v", err)
	}

	if len(report.Categories) == 0 {
		t.Fatal("expected diagnostic categories, got 0")
	}

	// Verify JSON serialization
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report to JSON: %v", err)
	}

	var decoded doctor.DiagnosticReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON into DiagnosticReport: %v", err)
	}

	if decoded.Overall == "" {
		t.Error("expected overall status to be set")
	}
}
