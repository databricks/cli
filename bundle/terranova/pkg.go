package terranova

import (
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

type DeploymentUnit struct {
	Group      string
	Key        string
	Adapter    *tnresources.Adapter
	ActionType deployplan.ActionType
}

type DeploymentBundle struct {
	StateDB   tnstate.TerranovaState
	Graph     *dagrun.Graph[deployplan.ResourceNode]
	Resources map[deployplan.ResourceNode]DeploymentUnit
	Adapters  map[string]*tnresources.Adapter
}
