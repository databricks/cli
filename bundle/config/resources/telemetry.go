package resources

import "github.com/databricks/cli/libs/telemetry/protos"

// LifecycleStartedTelemetry is implemented by resource types that support
// lifecycle.started. Deploy telemetry reports lifecycle.started of a resource of the
// type that sets it, under the key the type declares.
type LifecycleStartedTelemetry interface {
	// LifecycleStartedTelemetryKey returns the key of the deploy telemetry value that
	// reports lifecycle.started for this type.
	LifecycleStartedTelemetryKey() string
}

// DeployTelemetryIDs is implemented by resource types whose IDs the deploy telemetry
// event lists in a field of its own.
type DeployTelemetryIDs interface {
	// DeployTelemetryIDsField returns the field of event that lists the IDs of resources
	// of this type.
	DeployTelemetryIDsField(event *protos.BundleDeployEvent) *[]string

	// GetID returns the ID of the deployed resource, or "" if it is not deployed.
	GetID() string
}
