package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/require"
)

// The suite reuses the acceptance invariant corpus rather than growing a second set
// of bundle fixtures: those configs are already known-deployable and maintained.
const (
	invariantDir = "../../../acceptance/bundle/invariant"
	configsDir   = invariantDir + "/configs"
	dataDir      = invariantDir + "/data"
)

// testConfig is one usable entry of the corpus.
type testConfig struct {
	name         string // "schema.yml.tmpl"
	resourceType string // "schemas"
}

// discoverConfigs lists the corpus entries this suite can drive today, and returns the
// rest with the reason they were left out so the report can account for every config.
//
// Out of scope for now:
//   - an -init.sh counterpart: the config needs shell setup this suite does not run
//   - more than one resource: field edits would need the whole dependency graph
//   - permissions and grants sub-resources: they are separate plan nodes whose fields
//     describe an ACL rather than the resource, and they need a parent to attach to
func discoverConfigs(t *testing.T) (usable []testConfig, skipped map[string]string) {
	entries, err := filepath.Glob(filepath.Join(configsDir, "*.yml.tmpl"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	skipped = map[string]string{}
	seen := map[string]string{}
	for _, path := range entries {
		name := filepath.Base(path)

		if _, err := os.Stat(path + "-init.sh"); err == nil {
			skipped[name] = "has -init.sh"
			continue
		}

		nodes, err := resourceNodes(path)
		if err != nil {
			skipped[name] = "unparseable: " + err.Error()
			continue
		}
		if len(nodes) != 1 {
			skipped[name] = "has " + strconv.Itoa(len(nodes)) + " resources"
			continue
		}
		resourceType := bundleconfig.GetResourceTypeFromKey(nodes[0])
		if _, ok := dresources.SupportedResources[resourceType]; !ok {
			skipped[name] = "unsupported resource type " + resourceType
			continue
		}

		// The suite strips permissions and grants, so two configs differing only in
		// those blocks would test exactly the same fields twice.
		fingerprint, err := resourceFingerprint(path)
		if err != nil {
			skipped[name] = "unfingerprintable: " + err.Error()
			continue
		}
		if first, dup := seen[fingerprint]; dup {
			skipped[name] = "same resource as " + first + " once permissions and grants are removed"
			continue
		}
		seen[fingerprint] = name

		usable = append(usable, testConfig{name: name, resourceType: resourceType})
	}

	slices.SortFunc(usable, func(a, b testConfig) int { return strings.Compare(a.name, b.name) })
	return usable, skipped
}

// resourceFingerprint identifies the resources a config declares, as this suite will
// actually deploy them: sub-resources removed, and the bundle block ignored because it
// carries only the run's unique name.
func resourceFingerprint(path string) (string, error) {
	root, err := loadTemplate(path)
	if err != nil {
		return "", err
	}
	stripped, err := withoutSubResources(root)
	if err != nil {
		return "", err
	}
	resources, err := dyn.Get(stripped, "resources")
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(resources.AsAny())
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// loadTemplate parses a config template with its variables blanked out; only the shape
// matters to the callers here.
func loadTemplate(path string) (dyn.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return dyn.InvalidValue, err
	}
	yml := os.Expand(string(data), func(string) string { return "x" })
	return yamlloader.LoadYAML(path, strings.NewReader(yml))
}

// resourceNodes returns the "resources.<type>.<name>" keys declared by a config
// template, ignoring the permissions and grants sub-blocks.
func resourceNodes(path string) ([]string, error) {
	root, err := loadTemplate(path)
	if err != nil {
		return nil, err
	}

	var nodes []string
	_, err = dyn.MapByPattern(root,
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			nodes = append(nodes, p.String())
			return v, nil
		})
	if err != nil {
		return nil, err
	}
	slices.Sort(nodes)
	return nodes, nil
}

// soleResourceNode returns the single resource node of an initialized config.
func soleResourceNode(root *bundleconfig.Root) (string, error) {
	nodes, err := initializedNodes(root)
	if err != nil {
		return "", err
	}
	if len(nodes) != 1 {
		return "", fmt.Errorf("expected exactly one resource node, got %v", nodes)
	}
	return nodes[0], nil
}

func initializedNodes(root *bundleconfig.Root) ([]string, error) {
	var nodes []string
	_, err := dyn.MapByPattern(root.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			nodes = append(nodes, p.String())
			return v, nil
		})
	slices.Sort(nodes)
	return nodes, err
}
