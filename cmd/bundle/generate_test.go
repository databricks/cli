package bundle

import (
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateUnsupported lists the resource groups that deliberately have no generate
// subcommand. A group that is in neither this list nor the command tree fails the
// test, so every new resource type gets an explicit decision.
var generateUnsupported = []string{
	"internal_immutable_snapshots",
	"job_runs",
}

func TestGenerateCommandCoversAllResourceGroups(t *testing.T) {
	groups := config.SupportedResources()

	// Subcommands are named after the resource type's singular name with dashes.
	byCommand := make(map[string]string, len(groups))
	for group, d := range groups {
		byCommand[strings.ReplaceAll(d.SingularName, "_", "-")] = group
	}

	covered := make(map[string]bool)
	for _, sub := range newGenerateCommand().Commands() {
		group, ok := byCommand[sub.Name()]
		require.True(t, ok, "generate subcommand %q does not match the singular name of any resource type", sub.Name())
		covered[group] = true
	}

	for _, group := range generateUnsupported {
		assert.Contains(t, groups, group, "%q is not a resource group; drop it from generateUnsupported", group)
		assert.False(t, covered[group], "%q has a generate subcommand; drop it from generateUnsupported", group)
	}
	for group := range groups {
		if slices.Contains(generateUnsupported, group) {
			continue
		}
		assert.True(t, covered[group], "%q has no generate subcommand; add one or list it in generateUnsupported", group)
	}
}
