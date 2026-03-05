package main

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPathMapping(t *testing.T) {
	m := buildPathMapping()

	// Verify some well-known mappings
	assert.Equal(t, "bundle", m.typeToBundlePath["github.com/databricks/cli/bundle/config.Bundle"])
	assert.Equal(t, "workspace", m.typeToBundlePath["github.com/databricks/cli/bundle/config.Workspace"])
	assert.Equal(t, "resources", m.typeToBundlePath["github.com/databricks/cli/bundle/config.Resources"])
	assert.Equal(t, "resources.jobs.*", m.typeToBundlePath["github.com/databricks/cli/bundle/config/resources.Job"])
	assert.Equal(t, "resources.pipelines.*", m.typeToBundlePath["github.com/databricks/cli/bundle/config/resources.Pipeline"])
	assert.Equal(t, "resources.clusters.*", m.typeToBundlePath["github.com/databricks/cli/bundle/config/resources.Cluster"])
	assert.Equal(t, "variables.*", m.typeToBundlePath["github.com/databricks/cli/bundle/config/variable.Variable"])

	// Reverse mapping
	assert.Equal(t, "github.com/databricks/cli/bundle/config.Bundle", m.bundlePathToType["bundle"])
	assert.Equal(t, "github.com/databricks/cli/bundle/config/resources.Job", m.bundlePathToType["resources.jobs.*"])
}

func TestPathMappingCoversAllAnnotatedTypes(t *testing.T) {
	m := buildPathMapping()

	annotations, err := getAnnotations("annotations.yml")
	assert.NoError(t, err)

	var unmapped []string
	for typePath := range annotations {
		// Skip keys that are already bundle paths (after conversion).
		if _, ok := m.bundlePathToType[typePath]; ok {
			continue
		}
		if _, ok := m.typeToBundlePath[typePath]; !ok {
			unmapped = append(unmapped, typePath)
		}
	}

	if len(unmapped) > 0 {
		slices.Sort(unmapped)
		for _, u := range unmapped {
			fmt.Printf("Unmapped: %s\n", u)
		}
	}
	assert.Empty(t, unmapped, "All annotated types should have a bundle path mapping")
}
