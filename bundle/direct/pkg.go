package direct

import (
	"context"
	"errors"
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
	StateDB          dstate.DeploymentState
	Adapters         map[string]*dresources.Adapter
	Plan             *deployplan.Plan
	RemoteStateCache sync.Map
	StateCache       structvar.Cache

	// deployErrs collects errors reported by the parallel graph workers in
	// CalculatePlan and Apply. Workers record errors here (thread-safe) and
	// signal failure to the graph by returning false; the collected errors are
	// joined and returned to the caller once the run completes.
	deployErrs errorList
}

// errorList is a thread-safe accumulator for errors produced by parallel workers.
type errorList struct {
	mu   sync.Mutex
	errs []error
}

func (e *errorList) reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errs = nil
}

func (e *errorList) add(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errs = append(e.errs, err)
}

func (e *errorList) join() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return errors.Join(e.errs...)
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
