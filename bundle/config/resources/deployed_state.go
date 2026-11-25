package resources

import "github.com/databricks/cli/bundle/statemgmt/resourcestate"

// DeployedStateLoader is implemented by resource types whose config takes values that only the
// deployed state has.
type DeployedStateLoader interface {
	// LoadDeployedState copies such values from the resource's exported state into the typed
	// config. It writes the typed config only, so the dynamic config, including references that
	// carry plan dependencies, stays as written.
	LoadDeployedState(state resourcestate.ResourceState)
}
