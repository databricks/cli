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
			return !(schemaFails && key == schemaKey)
		})

		if schemaFails {
			assert.Equal(t, []string{volumeKey, schemaKey}, order)
			assert.Equal(t, map[string]bool{modelKey: true}, failed)
		} else {
			assert.Equal(t, []string{volumeKey, schemaKey, modelKey}, order)
		}
	}
}
