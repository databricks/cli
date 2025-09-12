package direct

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

// DeploymentUnit holds state + adapter (implementation) for a single resource
type DeploymentUnit struct {
	// Resource identifier: "resources.jobs.foo" or "resources.jobs.foo.permissions"
	ResourceKey string

	// Implementation for this resource; all deployments from the same group share the adapter
	Adapter *dresources.Adapter

	// Planned ActionType
	ActionType deployplan.ActionType

	// Remote state (pointer to adapter.RemoteType()) or nil if remote state was not fetched yet.
	// Remote state will be eagerly populated by (withRefresh) DoCreate/DoUpdate/WaitForCreate/WaitForUpdate.
	// If the resource does not implement withRefresh variants of those methods, remoteState remains nil and
	// will be populated lazily by calling DoRefresh().
	RemoteState any
}

// DeploymentBundle holds everything needed to deploy a bundle
type DeploymentBundle struct {
	StateDB          dstate.DeploymentState
	Adapters         map[string]*dresources.Adapter
	Plan             *deployplan.Plan
	RemoteStateCache map[string]any
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
	return nil
}
