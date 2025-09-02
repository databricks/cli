package terranova

import (
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

// DeploymentUnit holds state + adapter (implementation) for a single resource
type DeploymentUnit struct {
	Group      string
	Key        string
	Adapter    *tnresources.Adapter
	ActionType deployplan.ActionType
}

// DeploymentBundle holds everything needed to deploy a bundle
type DeploymentBundle struct {
	StateDB         tnstate.TerranovaState
	Graph           *dagrun.Graph[deployplan.ResourceNode]
	DeploymentUnits map[deployplan.ResourceNode]DeploymentUnit
	Adapters        map[string]*tnresources.Adapter
}
