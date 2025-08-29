package terranova

import (
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

type Deployer struct {
	Group       string
	Key         string
	Resource    IResource
	Fresh       bool
	ActionType  deployplan.ActionType
	RemoteState any // Tracks the current remote state from API calls
}

type BundleDeployer struct {
	StateDB   tnstate.TerranovaState
	Graph     *dagrun.Graph[deployplan.ResourceNode]
	Resources map[deployplan.ResourceNode]Deployer
}
