package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportedNamesMatchesRegistry(t *testing.T) {
	names := SupportedNames()
	assert.Len(t, names, len(Registry))
	for i, a := range Registry {
		assert.Equal(t, a.DisplayName, names[i])
	}
}

func TestSkillsOnlyNamesMatchesRegistry(t *testing.T) {
	names := SkillsOnlyNames()
	// Skills-only agents (Plugin nil) are listed; plugin agents are not.
	assert.Contains(t, names, "Pi")
	assert.Contains(t, names, "Goose")
	assert.NotContains(t, names, "Claude Code")
	for _, a := range Registry {
		if a.Plugin != nil {
			assert.NotContains(t, names, a.DisplayName)
		}
	}
}
