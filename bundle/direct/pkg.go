package direct

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/structs/structvar"
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
	// will be populated lazily by calling DoRead().
	RemoteState any

	// DependsOn lists resources this resource depends on (persisted in state).
	DependsOn []deployplan.DependsOnEntry
}

// DeploymentBundle holds everything needed to deploy a bundle
type DeploymentBundle struct {
	StateDB           dstate.DeploymentState
	Adapters          map[string]*dresources.Adapter
	InternalResources []InternalResource
	Plan              *deployplan.Plan
	RemoteStateCache  sync.Map
	StateCache        structvar.Cache
}

type InternalResource struct {
	Key         string
	Adapter     *dresources.Adapter
	InputConfig any // passed to adapter.PrepareState during makePlan
}

// RegisterInternalResource creates an adapter from instance and queues the
// resource for planning and apply. Must be called before CalculatePlan.
func (b *DeploymentBundle) RegisterInternalResource(key string, instance any, inputConfig any) error {
	adapter, err := dresources.NewAdapterFromInstance(instance, key)
	if err != nil {
		return fmt.Errorf("registering internal resource %s: %w", key, err)
	}
	b.InternalResources = append(b.InternalResources, InternalResource{
		Key:         key,
		Adapter:     adapter,
		InputConfig: inputConfig,
	})
	return nil
}

func (b *DeploymentBundle) findInternalAdapter(key string) *dresources.Adapter {
	for i := range b.InternalResources {
		if b.InternalResources[i].Key == key {
			return b.InternalResources[i].Adapter
		}
	}
	return nil
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

// ExportState exports the current deployment state as a resource map.
// StateDB must already be open for read before calling this function.
func (b *DeploymentBundle) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	b.StateDB.AssertOpenedForRead()
	return b.StateDB.ExportState(ctx)
}
