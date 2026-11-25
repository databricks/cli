package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	invariantConfigsDir   = "bundle/invariant/configs"
	invariantConfigSuffix = ".yml.tmpl"
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

// invariantConfigsNotAboutOneType lists the configs in invariantConfigsDir that exercise
// several resource types together and therefore are not named after one of them.
var invariantConfigsNotAboutOneType = map[string]bool{
	"uc_trailing_slash.yml.tmpl": true,
}

// TestInvariantConfigsCoverage ensures that the invariant test configs in
// bundle/invariant/configs cover every bundle resource type, and that resource
// types supporting permissions or grants have at least one config exercising them.
//
// config.ResourcesTypes is the source of truth: it maps each resource group
// (e.g. "jobs") to its Go type and, where the resource struct has a Permissions
// or Grants field, adds derived keys "<group>.permissions" and "<group>.grants".
// A resource group is covered by the config named after it, "<group>.yml.tmpl".
func TestInvariantConfigsCoverage(t *testing.T) {
	groupsByConfig, withPermissions, withGrants := scanInvariantConfigs(t)

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
			covered = groupsByConfig[key+invariantConfigSuffix][key]
			hint = "named " + key + invariantConfigSuffix + " defines a " + key + " resource"
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

// TestInvariantConfigNames ensures that every config in invariantConfigsDir is named after
// the resource group it is about: "<group>.yml.tmpl", or "<group>_<variant>.yml.tmpl".
func TestInvariantConfigNames(t *testing.T) {
	names := invariantConfigNames(t)

	for _, name := range names {
		if invariantConfigsNotAboutOneType[name] {
			continue
		}
		base := strings.TrimSuffix(name, invariantConfigSuffix)
		named := false
		for key := range config.ResourcesTypes() {
			if strings.Contains(key, ".") {
				continue
			}
			if base == key || strings.HasPrefix(base, key+"_") {
				named = true
				break
			}
		}
		assert.True(t, named,
			"%s/%s is not named after a key under resources: (<key>%s or <key>_<variant>%s); rename it or list it in invariantConfigsNotAboutOneType",
			invariantConfigsDir, name, invariantConfigSuffix, invariantConfigSuffix)
	}

	for name := range invariantConfigsNotAboutOneType {
		assert.Contains(t, names, name, "remove %q from invariantConfigsNotAboutOneType", name)
	}
}

// scanInvariantConfigs parses every config in the invariant configs directory and
// returns the resource groups each config defines, the groups with at least one resource
// carrying permissions, and the groups with at least one resource carrying grants.
func scanInvariantConfigs(t *testing.T) (groupsByConfig map[string]map[string]bool, withPermissions, withGrants map[string]bool) {
	groupsByConfig = map[string]map[string]bool{}
	withPermissions = map[string]bool{}
	withGrants = map[string]bool{}

	for _, name := range invariantConfigNames(t) {
		path := filepath.Join(invariantConfigsDir, name)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)

		v, err := yamlloader.LoadYAML(path, strings.NewReader(string(contents)))
		require.NoError(t, err, "failed to parse %s", path)

		groups := map[string]bool{}
		groupsByConfig[name] = groups

		resources := v.Get("resources")
		if resources.Kind() != dyn.KindMap {
			// Some configs (e.g. PyDABs) declare resources outside of YAML.
			continue
		}

		for _, group := range resources.MustMap().Pairs() {
			groupName := group.Key.MustString()
			groups[groupName] = true

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

	return groupsByConfig, withPermissions, withGrants
}

// invariantConfigNames returns the file names of the templates in invariantConfigsDir.
func invariantConfigNames(t *testing.T) []string {
	entries, err := os.ReadDir(invariantConfigsDir)
	require.NoError(t, err)

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), invariantConfigSuffix) {
			continue
		}
		names = append(names, entry.Name())
	}
	return names
}
