package terranova

import (
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
)

type Deployer struct {
	Group      string
	Key        string
	Resource   IResource
	Fresh      bool
	ActionType deployplan.ActionType
}

type BundleDeployer struct {
	StateDB   tnstate.TerranovaState
	Graph     *dagrun.Graph[deployplan.ResourceNode]
	Resources map[deployplan.ResourceNode]Deployer
}

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

/*

# Vocabulary

## input config

Refers to resource config as defined in bundle/config/resources package.

Example: (bundle/config/resources).Job.

## snapshot

"snapshot" refers to processed config that we actually use to compare the differences. It is also what we store in the state.
The type of snapshot is typically a subset of input config.

Example: (go sdk/jobs).JobSettings.

## remote

Refers to state we fetched from the backend. This type is typically a superset of snapshot. Sometimes it's the same exact type, sometimes a different type.

# "SnapshotType", "RemoteType"



*/
