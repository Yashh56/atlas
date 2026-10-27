package skills_test

import (
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/skills"
)

func TestDefaultFixBuildSkill(t *testing.T) {
	if skills.DefaultFixBuildSkill == "" {
		t.Fatal("expected DefaultFixBuildSkill to be non-empty")
	}

	for _, keyword := range []string{"Fix Build Skill", "old_str", "new_str", "reasoning"} {
		if !strings.Contains(skills.DefaultFixBuildSkill, keyword) {
			t.Errorf("expected embedded skill to contain %q", keyword)
		}
	}
}
