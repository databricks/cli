package autotest

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/stretchr/testify/require"
)

// dataDir holds the files fixtures reference by path -- an app's source directory, a
// dashboard's serialized definition, a pipeline's notebook. These are shared test assets rather
// than resource definitions, so they stay where every suite reads them from; the fixtures
// themselves live in testdata/fields and belong to this suite alone.
const dataDir = "../../../acceptance/bundle/invariant/data"

// discoverFixtures returns the resource types this suite drives -- one value library each --
// and the supported types that have none, so the report accounts for every type the direct
// engine knows rather than only the ones covered.
//
// Sub-resources are not among either: permissions and grants are separate plan nodes whose
// fields describe an ACL rather than the resource, they need a parent to attach to, and the
// suite strips them from every fixture. Listing them as gaps would suggest fixtures are missing.
func discoverFixtures(t *testing.T) (driven, undriven []string) {
	entries, err := filepath.Glob(filepath.Join(fieldsDir, "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	has := map[string]bool{}
	for _, path := range entries {
		resourceType := strings.TrimSuffix(filepath.Base(path), ".yml")
		// A library for a type the engine does not support is a leftover, and silently
		// ignoring it would let a rename go unnoticed.
		_, supported := dresources.SupportedResources[resourceType]
		require.True(t, supported, "%s names no supported resource type", path)
		has[resourceType] = true
		driven = append(driven, resourceType)
	}

	for resourceType := range dresources.SupportedResources {
		if !has[resourceType] && !strings.Contains(resourceType, ".") {
			undriven = append(undriven, resourceType)
		}
	}

	slices.Sort(driven)
	slices.Sort(undriven)
	return driven, undriven
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
