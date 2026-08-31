package autotest

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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
	for _, resource := range declaredResources(root) {
		for _, kind := range subResourceKinds {
			delete(resource, kind)
		}
	}
	encoded, err := json.Marshal(root["resources"])
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// loadTemplate parses a config template with its variables blanked out; only the shape
// matters to the callers here.
func loadTemplate(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	yml := os.Expand(string(data), func(string) string { return "x" })
	var root map[string]any
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return root, nil
}

// resourceNodes returns the "resources.<type>.<name>" keys declared by a config template.
func resourceNodes(path string) ([]string, error) {
	root, err := loadTemplate(path)
	if err != nil {
		return nil, err
	}
	var nodes []string
	for _, group := range sortedKeys(mapAt(root, "resources")) {
		for _, name := range sortedKeys(mapAt(mapAt(root, "resources"), group)) {
			nodes = append(nodes, "resources."+group+"."+name)
		}
	}
	return nodes, nil
}

// declaredResources returns every resource body of a parsed template.
func declaredResources(root map[string]any) []map[string]any {
	var out []map[string]any
	resources := mapAt(root, "resources")
	for _, group := range sortedKeys(resources) {
		byName := mapAt(resources, group)
		for _, name := range sortedKeys(byName) {
			if body, ok := byName[name].(map[string]any); ok {
				out = append(out, body)
			}
		}
	}
	return out
}

func mapAt(parent map[string]any, key string) map[string]any {
	value, _ := parent[key].(map[string]any)
	return value
}

func sortedKeys(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
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
	err := forEachResource(root, func(node string, _ any) error {
		nodes = append(nodes, node)
		return nil
	})
	slices.Sort(nodes)
	return nodes, err
}
