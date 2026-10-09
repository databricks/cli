package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

const invariantConfigsDir = "bundle/invariant/configs"

// LackingInvariantTest lists keys from config.ResourcesTypes that knowingly lack
// a covering config in invariantConfigsDir. Keys match the ResourcesTypes
// form: "<group>" for the resource itself, "<group>.permissions" / "<group>.grants"
// for permissions/grants coverage. Add a config and remove the entry to close a gap;
// the test fails if an entry here is actually covered, so the list only shrinks.
var LackingInvariantTest = map[string]bool{
	"quality_monitors":             true,
	"internal_immutable_snapshots": true,
}

// TestInvariantConfigsCoverage ensures that the invariant test configs in
// bundle/invariant/configs cover every bundle resource type, and that resource
// types supporting permissions or grants have at least one config exercising them.
//
// config.ResourcesTypes is the source of truth: it maps each resource group
// (e.g. "jobs") to its Go type and, where the resource struct has a Permissions
// or Grants field, adds derived keys "<group>.permissions" and "<group>.grants".
func TestInvariantConfigsCoverage(t *testing.T) {
	present, withPermissions, withGrants := scanInvariantConfigs(t)

	keys := make([]string, 0, len(config.ResourcesTypes))
	for key := range config.ResourcesTypes {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		var covered bool
		var hint string
		switch {
		case strings.HasSuffix(key, ".permissions"):
			group := strings.TrimSuffix(key, ".permissions")
			covered = withPermissions[group]
			hint = "attaches permissions to a " + group + " resource"
		case strings.HasSuffix(key, ".grants"):
			group := strings.TrimSuffix(key, ".grants")
			covered = withGrants[group]
			hint = "attaches grants to a " + group + " resource"
		default:
			covered = present[key]
			hint = "defines a " + key + " resource"
		}

		if LackingInvariantTest[key] {
			assert.False(t, covered,
				"%q is covered by a config in %s; remove it from LackingInvariantTest", key, invariantConfigsDir)
			continue
		}
		assert.True(t, covered,
			"no config in %s %s; add one or allowlist %q", invariantConfigsDir, hint, key)
	}
}

// scanInvariantConfigs parses every config in the invariant configs directory and
// returns the set of resource groups present, the groups with at least one resource
// carrying permissions, and the groups with at least one resource carrying grants.
func scanInvariantConfigs(t *testing.T) (present, withPermissions, withGrants map[string]bool) {
	present = map[string]bool{}
	withPermissions = map[string]bool{}
	withGrants = map[string]bool{}

	entries, err := os.ReadDir(invariantConfigsDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml.tmpl") {
			continue
		}
		path := filepath.Join(invariantConfigsDir, entry.Name())
		contents, err := os.ReadFile(path)
		require.NoError(t, err)

		var doc struct {
			Resources map[string]any `yaml:"resources"`
		}
		require.NoError(t, yaml.Unmarshal(contents, &doc), "failed to parse %s", path)

		// Some configs (e.g. PyDABs) declare resources outside of YAML.
		for groupName, group := range doc.Resources {
			present[groupName] = true

			resources, _ := group.(map[string]any)
			for _, resource := range resources {
				cfg, _ := resource.(map[string]any)
				if _, ok := cfg["permissions"]; ok {
					withPermissions[groupName] = true
				}
				if _, ok := cfg["grants"]; ok {
					withGrants[groupName] = true
				}
			}
		}
	}

	return present, withPermissions, withGrants
}
