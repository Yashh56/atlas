package updater

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input      string
		wantMaj    int
		wantMin    int
		wantPatch  int
		wantSuffix string
		wantOk     bool
	}{
		{"v0.4.0", 0, 4, 0, "", true},
		{"0.4.0", 0, 4, 0, "", true},
		{"v0.4.0-2-gf774455-dirty", 0, 4, 0, "-2-gf774455-dirty", true},
		{"v0.4.1", 0, 4, 1, "", true},
		{"v1.2.3-beta.1", 1, 2, 3, "-beta.1", true},
		{"v1.2.3+build123", 1, 2, 3, "+build123", true},
		{"dev", 0, 0, 0, "", false},
		{"invalid", 0, 0, 0, "", false},
		{"v1.2", 0, 0, 0, "", false},
	}

	for _, tc := range tests {
		maj, min, patch, suffix, ok := parseSemver(tc.input)
		if ok != tc.wantOk {
			t.Errorf("parseSemver(%q) ok = %v, want %v", tc.input, ok, tc.wantOk)
			continue
		}
		if !ok {
			continue
		}
		if maj != tc.wantMaj || min != tc.wantMin || patch != tc.wantPatch || suffix != tc.wantSuffix {
			t.Errorf("parseSemver(%q) = (%d, %d, %d, %q), want (%d, %d, %d, %q)",
				tc.input, maj, min, patch, suffix, tc.wantMaj, tc.wantMin, tc.wantPatch, tc.wantSuffix)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v0.4.0", "v0.4.0-2-gf774455-dirty", false},
		{"v0.4.1", "v0.4.0-2-gf774455-dirty", true},
		{"v0.4.0", "v0.4.0", false},
		{"v0.4.0", "v0.3.0", true},
		{"v0.4.0", "dev", false},
		{"v1.0.0", "dev", false},
		{"v1.0.0", "v0.9.9", true},
		{"v0.4.0", "v0.5.0", false},
		{"v0.4.0", "v0.4.1", false},
		{"v1.0.0-rc1", "v1.0.0-rc1", false},
	}

	for _, tc := range tests {
		got := isNewer(tc.latest, tc.current)
		if got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}
