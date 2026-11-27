package direct

import (
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeGraphDeletesWithParentAfterItsParent(t *testing.T) {
	const (
		schemaKey = "resources.schemas.schema1"
		volumeKey = "resources.volumes.volume1"
		modelKey  = "resources.registered_models.model1"
	)
	schemaDep := deployplan.DependsOnEntry{Node: schemaKey, Label: "${resources.schemas.schema1.name}"}
	plan := deployplan.NewPlanDirect()
	plan.Plan[schemaKey] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved"}
	// Verified under the moved schema: deleted as itself, before the schema.
	plan.Plan[volumeKey] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved.v", DependsOn: []deployplan.DependsOnEntry{schemaDep}}
	// Goes with the schema's delete: its state is dropped only once that delete succeeded.
	plan.Plan[modelKey] = &deployplan.PlanEntry{Action: deployplan.Delete, Gone: true, WithParent: schemaKey, DependsOn: []deployplan.DependsOnEntry{schemaDep}}

	for _, schemaFails := range []bool{false, true} {
		g, err := makeGraph(plan)
		require.NoError(t, err)
		require.NoError(t, g.DetectCycle())

		var mu sync.Mutex
		var order []string
		failed := map[string]bool{}
		g.Run(1, func(key string, failedDependency *string) bool {
			mu.Lock()
			defer mu.Unlock()
			if failedDependency != nil {
				failed[key] = true
				return false
			}
			order = append(order, key)
			return !schemaFails || key != schemaKey
		})

		if schemaFails {
			assert.Equal(t, []string{volumeKey, schemaKey}, order)
			assert.Equal(t, map[string]bool{modelKey: true}, failed)
		} else {
			assert.Equal(t, []string{volumeKey, schemaKey, modelKey}, order)
		}
	}
}

func TestMakeGraphWithParentReferencingASibling(t *testing.T) {
	const (
		schemaKey  = "resources.schemas.schema1"
		volume1Key = "resources.volumes.volume1"
		volume2Key = "resources.volumes.volume2"
	)
	plan := deployplan.NewPlanDirect()
	plan.Plan[schemaKey] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved"}
	plan.Plan[volume1Key] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved.v1", DependsOn: []deployplan.DependsOnEntry{{Node: schemaKey}}}
	// volume2 names its schema through volume1: deleted before volume1 as usual, it would close a
	// cycle through the schema it goes with.
	plan.Plan[volume2Key] = &deployplan.PlanEntry{
		Action: deployplan.Delete, Gone: true, WithParent: schemaKey,
		DependsOn: []deployplan.DependsOnEntry{{Node: volume1Key}, {Node: schemaKey}},
	}

	g, err := makeGraph(plan)
	require.NoError(t, err)
	require.NoError(t, g.DetectCycle())

	var order []string
	g.Run(1, func(key string, failedDependency *string) bool {
		order = append(order, key)
		return true
	})
	assert.Equal(t, []string{volume1Key, schemaKey, volume2Key}, order)
}

func TestMakeGraphDependentOfWithParentWaitsForItsParent(t *testing.T) {
	const (
		schemaKey = "resources.schemas.schema1"
		volumeKey = "resources.volumes.volume1"
		jobKey    = "resources.jobs.job1"
	)
	plan := deployplan.NewPlanDirect()
	plan.Plan[schemaKey] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved"}
	plan.Plan[volumeKey] = &deployplan.PlanEntry{Action: deployplan.Delete, Gone: true, WithParent: schemaKey, DependsOn: []deployplan.DependsOnEntry{{Node: schemaKey}}}
	// The job names only the volume, whose removal is the schema's delete.
	plan.Plan[jobKey] = &deployplan.PlanEntry{Action: deployplan.Delete, DependsOn: []deployplan.DependsOnEntry{{Node: volumeKey}}}

	for _, jobFails := range []bool{false, true} {
		g, err := makeGraph(plan)
		require.NoError(t, err)
		require.NoError(t, g.DetectCycle())

		var mu sync.Mutex
		var order []string
		failed := map[string]bool{}
		g.Run(1, func(key string, failedDependency *string) bool {
			mu.Lock()
			defer mu.Unlock()
			if failedDependency != nil {
				failed[key] = true
				return false
			}
			order = append(order, key)
			return !jobFails || key != jobKey
		})

		if jobFails {
			assert.Equal(t, []string{jobKey}, order)
			assert.Equal(t, map[string]bool{schemaKey: true, volumeKey: true}, failed)
		} else {
			assert.Equal(t, []string{jobKey, schemaKey, volumeKey}, order)
		}
	}
}

func TestMakeGraphParentOfWithParentWaitsForItsDependencies(t *testing.T) {
	const (
		locationKey = "resources.external_locations.loc"
		schemaKey   = "resources.schemas.schema1"
		volumeKey   = "resources.volumes.volume1"
	)
	plan := deployplan.NewPlanDirect()
	plan.Plan[locationKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
	plan.Plan[schemaKey] = &deployplan.PlanEntry{Action: deployplan.Delete, ID: "main.moved"}
	// An external volume whose storage_location names the location: the schema's delete removes it.
	plan.Plan[volumeKey] = &deployplan.PlanEntry{
		Action: deployplan.Delete, Gone: true, WithParent: schemaKey,
		DependsOn: []deployplan.DependsOnEntry{{Node: schemaKey}, {Node: locationKey}},
	}

	for _, schemaFails := range []bool{false, true} {
		g, err := makeGraph(plan)
		require.NoError(t, err)
		require.NoError(t, g.DetectCycle())

		var mu sync.Mutex
		var order []string
		failed := map[string]bool{}
		g.Run(1, func(key string, failedDependency *string) bool {
			mu.Lock()
			defer mu.Unlock()
			if failedDependency != nil {
				failed[key] = true
				return false
			}
			order = append(order, key)
			return !schemaFails || key != schemaKey
		})

		if schemaFails {
			assert.Equal(t, []string{schemaKey}, order)
			assert.Equal(t, map[string]bool{locationKey: true, volumeKey: true}, failed)
		} else {
			assert.Equal(t, []string{schemaKey, volumeKey, locationKey}, order)
		}
	}
}
