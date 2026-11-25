package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

const (
	resourceTestsDir = "bundle/resources"
	bindTestsDir     = "bundle/deployment/bind"
)

// snapshotsNotUserDeclared is the reason internal_immutable_snapshots has no test directories.
const snapshotsNotUserDeclared = "internal resource created by the deploy pipeline; it is not declared in databricks.yml"

// LackingResourceTests lists resource keys that knowingly have no directory
// <resourceTestsDir>/<key>, with the reason. Add the tests and remove the entry to close
// a gap; the test fails if a listed key has the directory, so the list only shrinks.
var LackingResourceTests = map[string]string{
	"internal_immutable_snapshots": snapshotsNotUserDeclared,
}

// LackingBindTests lists resource keys that knowingly have no directory
// <bindTestsDir>/<key>, with the reason. Same rules as LackingResourceTests.
var LackingBindTests = map[string]string{
	"apps":                         "not covered today",
	"cluster_policies":             "not covered today",
	"database_catalogs":            "not covered today",
	"instance_pools":               "not covered today",
	"internal_immutable_snapshots": snapshotsNotUserDeclared,
	"job_runs":                     "not covered today",
	"materialization_sets":         "not covered today",
	"models":                       "not covered today",
	"postgres_branches":            "not covered today",
	"postgres_catalogs":            "not covered today",
	"postgres_endpoints":           "not covered today",
	"postgres_projects":            "not covered today",
	"postgres_synced_tables":       "not covered today",
	"secrets":                      "not covered today",
	"synced_database_tables":       "not covered today",
}

// TestResourceTestDirsCoverage ensures every resource type in config.Resources has its
// acceptance test directories, named after its key under resources: in databricks.yml.
func TestResourceTestDirsCoverage(t *testing.T) {
	keys := resourceKeys()
	for _, key := range keys {
		checkResourceTestDir(t, resourceTestsDir, "LackingResourceTests", LackingResourceTests, key)
		checkResourceTestDir(t, bindTestsDir, "LackingBindTests", LackingBindTests, key)
	}

	for key := range LackingResourceTests {
		assert.True(t, slices.Contains(keys, key), "LackingResourceTests entry %q is not a resource type; remove it", key)
	}
	for key := range LackingBindTests {
		assert.True(t, slices.Contains(keys, key), "LackingBindTests entry %q is not a resource type; remove it", key)
	}
}

func checkResourceTestDir(t *testing.T, dir, listName string, lacking map[string]string, key string) {
	path := filepath.Join(dir, key)
	info, err := os.Stat(path)
	exists := err == nil && info.IsDir()

	if _, ok := lacking[key]; ok {
		assert.False(t, exists, "%q has tests in %s; remove it from %s", key, path, listName)
		return
	}
	assert.True(t, exists, "resource type %q has no tests in %s; add them or list %q in %s with the reason", key, path, key, listName)
}

// resourceKeys returns the sorted keys of the resource types, without the derived
// "<key>.permissions" and "<key>.grants" entries of config.ResourcesTypes.
func resourceKeys() []string {
	var keys []string
	for key := range config.ResourcesTypes() {
		if strings.Contains(key, ".") {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
