package terranova

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

// DeploymentUnit holds state + adapter (implementation) for a single resource
type DeploymentUnit struct {
	Group       string
	Key         string
	Adapter     *tnresources.Adapter
	ActionType  deployplan.ActionType
	Fresh       bool
	RemoteState any
}

// DeploymentBundle holds everything needed to deploy a bundle
type DeploymentBundle struct {
	StateDB         tnstate.TerranovaState
	Graph           *dagrun.Graph[deployplan.ResourceNode]
	DeploymentUnits map[deployplan.ResourceNode]*DeploymentUnit
	Adapters        map[string]*tnresources.Adapter
}

// SetRemoteState updates the remote state with type validation and marks as fresh.
// If remoteState is nil, no action is taken.
// If remoteState is not nil, it must match the expected RemoteType.
func (d *DeploymentUnit) SetRemoteState(remoteState any) error {
	if remoteState == nil {
		return nil
	}

	actualType := reflect.TypeOf(remoteState)
	remoteType := d.Adapter.RemoteType()
	if actualType != remoteType {
		return fmt.Errorf("internal error: remote state type mismatch: expected %s, got %s", remoteType, actualType)
	}

	d.RemoteState = remoteState
	d.Fresh = true
	return nil
}
