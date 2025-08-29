package terranova

import (
	"fmt"
	"reflect"

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
	RemoteState any          // Tracks the current remote state from API calls
	RemoteType  reflect.Type // Expected type for RemoteState validation
}

// SetRemoteState updates the remote state with type validation and marks as fresh.
// If remoteState is nil, no action is taken.
// If remoteState is not nil, it must match the expected RemoteType.
func (d *Deployer) SetRemoteState(remoteState any) error {
	if remoteState == nil {
		return nil
	}

	if d.RemoteType == nil {
		return fmt.Errorf("internal error: RemoteType not set for %s.%s", d.Group, d.Key)
	}

	actualType := reflect.TypeOf(remoteState)
	if actualType != d.RemoteType {
		return fmt.Errorf("remote state type mismatch for %s.%s: expected %s, got %s", d.Group, d.Key, d.RemoteType, actualType)
	}

	d.RemoteState = remoteState
	d.Fresh = true
	return nil
}

type BundleDeployer struct {
	StateDB   tnstate.TerranovaState
	Graph     *dagrun.Graph[deployplan.ResourceNode]
	Resources map[deployplan.ResourceNode]Deployer
}
