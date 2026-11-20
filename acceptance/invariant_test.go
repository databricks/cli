package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	invariantConfigsDir = "bundle/invariant/configs"
	invariantTestConfig = "bundle/invariant/test.toml"
)

// LackingInvariantTest lists keys from config.ResourcesTypes that knowingly lack
// a covering config in invariantConfigsDir. Keys match the ResourcesTypes
// form: "<group>" for the resource itself, "<group>.permissions" / "<group>.grants"
// for permissions/grants coverage. Add a config and remove the entry to close a gap;
// the test fails if an entry here is actually covered, so the list only shrinks.
var LackingInvariantTest = map[string]bool{
	"quality_monitors":             true,
	"internal_immutable_snapshots": true,
}

// UnlistedInvariantConfigs lists templates in invariantConfigsDir that are not in
// EnvMatrix["INPUT_CONFIG"] of invariantTestConfig, so the invariant suite never runs them.
// The list may only shrink: add a template to INPUT_CONFIG and remove it here to run it.
var UnlistedInvariantConfigs = map[string]bool{
	"app_lifecycle_ref.yml.tmpl":          true,
	"materialization_set.yml.tmpl":        true,
	"mcp_service.yml.tmpl":                true,
	"model_provider_service.yml.tmpl":     true,
	"postgres_snapshot_schedule.yml.tmpl": true,
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

	keys := make([]string, 0, len(config.ResourcesTypes()))
	for key := range config.ResourcesTypes() {
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

// TestInvariantConfigsListed ensures every template in invariantConfigsDir is run by the
// invariant matrix unless allowlisted in UnlistedInvariantConfigs, and that every matrix
// entry names an existing template. A template missing from the matrix is silently never run.
func TestInvariantConfigsListed(t *testing.T) {
	matrix := internal.DoLoadConfig(t, invariantTestConfig).EnvMatrix["INPUT_CONFIG"]
	listed := make(map[string]bool, len(matrix))
	for _, name := range matrix {
		listed[name] = true
	}

	entries, err := os.ReadDir(invariantConfigsDir)
	require.NoError(t, err)

	existing := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yml.tmpl") {
			continue
		}
		existing[name] = true

		if UnlistedInvariantConfigs[name] {
			assert.False(t, listed[name],
				"%q is in INPUT_CONFIG of %s; remove it from UnlistedInvariantConfigs", name, invariantTestConfig)
			continue
		}
		assert.True(t, listed[name],
			"%q in %s is not in INPUT_CONFIG of %s, so it never runs; add it or allowlist it in UnlistedInvariantConfigs",
			name, invariantConfigsDir, invariantTestConfig)
	}

	for _, name := range matrix {
		assert.True(t, existing[name], "INPUT_CONFIG entry %q has no template in %s", name, invariantConfigsDir)
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

		v, err := yamlloader.LoadYAML(path, strings.NewReader(string(contents)))
		require.NoError(t, err, "failed to parse %s", path)

		resources := v.Get("resources")
		if resources.Kind() != dyn.KindMap {
			// Some configs (e.g. PyDABs) declare resources outside of YAML.
			continue
		}

		for _, group := range resources.MustMap().Pairs() {
			groupName := group.Key.MustString()
			present[groupName] = true

			if group.Value.Kind() != dyn.KindMap {
				continue
			}
			for _, resource := range group.Value.MustMap().Pairs() {
				cfg := resource.Value
				if cfg.Kind() != dyn.KindMap {
					continue
				}
				if cfg.Get("permissions").Kind() != dyn.KindInvalid {
					withPermissions[groupName] = true
				}
				if cfg.Get("grants").Kind() != dyn.KindInvalid {
					withGrants[groupName] = true
				}
			}
		}
	}

	return present, withPermissions, withGrants
}
