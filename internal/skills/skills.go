package skills

import _ "embed"

// DefaultFixBuildSkill contains the default fix_build.md skill embedded into the binary.
//
//go:embed fix_build.md
var DefaultFixBuildSkill string
