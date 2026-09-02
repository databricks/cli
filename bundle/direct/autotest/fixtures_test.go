package autotest

import (
	"fmt"
	"os"
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

// ownDataDir holds files only this suite needs, staged next to the shared ones. Kept separate
// because everything in dataDir is uploaded by every invariant config too, so a file added there
// for one fixture changes those tests' output.
const ownDataDir = "testdata/data"

// drivenTypes returns every resource type the direct engine supports, which is what this suite
// covers: a type with no value library is a failure rather than a line in a report, so adding a
// resource type to the engine and forgetting the catalog cannot go unnoticed.
//
// Permissions and grants are excluded. They are separate plan nodes whose fields describe an ACL
// rather than the resource, they need a parent to attach to, and the suite strips them from every
// fixture.
func drivenTypes(t *testing.T) []string {
	var driven []string
	for resourceType := range dresources.SupportedResources {
		if strings.Contains(resourceType, ".") {
			continue
		}
		path := filepath.Join(fieldsDir, resourceType+".yml")
		_, err := os.Stat(path)
		require.NoError(t, err, "%s is supported by the engine but has no value library", resourceType)
		driven = append(driven, resourceType)
	}

	// A library naming no supported type is a leftover -- a rename that missed one side.
	entries, err := filepath.Glob(filepath.Join(fieldsDir, "*.yml"))
	require.NoError(t, err)
	for _, path := range entries {
		resourceType := strings.TrimSuffix(filepath.Base(path), ".yml")
		_, supported := dresources.SupportedResources[resourceType]
		require.True(t, supported, "%s names no supported resource type", path)
	}

	slices.Sort(driven)
	return driven
}

// resourceNode returns the node of the resource under test, which a fixture declares under
// resourceKey. A fixture may declare dependencies of other types alongside it, so this is not
// simply the only node in the config.
func resourceNode(root *bundleconfig.Root, resourceType string) (string, error) {
	want := "resources." + resourceType + "." + resourceKey
	nodes, err := initializedNodes(root)
	if err != nil {
		return "", err
	}
	if !slices.Contains(nodes, want) {
		return "", fmt.Errorf("%s is not among the declared resources %v", want, nodes)
	}
	return want, nil
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
