package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Yashh56/atlas/internal/doctor"
)

func TestDoctorCommandJSON(t *testing.T) {
	origJSON := jsonFlag
	defer func() { jsonFlag = origJSON }()

	jsonFlag = true
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"doctor", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("doctor execution failed: %v", err)
	}

	var report doctor.DiagnosticReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON DiagnosticReport, got parse error: %v, output: %s", err, buf.String())
	}

	if len(report.Categories) == 0 {
		t.Error("expected categories in diagnostic report")
	}
}
