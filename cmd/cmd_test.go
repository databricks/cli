package cmd

import (
	"testing"

	skillscmd "github.com/databricks/cli/cmd/skills"
	"github.com/stretchr/testify/assert"
)

func TestRootCommandNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	// The skills group is registered only behind DATABRICKS_SKILLS_PREVIEW, so
	// it is not in topLevelSpecs. It is checked here anyway, so a clash with a
	// workspace command is caught while the variable is off.
	specs := append(topLevelSpecs(), skillscmd.Spec)
	for _, spec := range specs {
		assert.False(t, seen[spec.Name()], "duplicate top-level command %q", spec.Name())
		seen[spec.Name()] = true
	}
}
