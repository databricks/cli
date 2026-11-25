package resources

// LifecycleStartedTelemetry is implemented by resource types that support
// lifecycle.started. Deploy telemetry reports lifecycle.started of a resource of the
// type that sets it, under the key the type declares.
type LifecycleStartedTelemetry interface {
	// LifecycleStartedTelemetryKey returns the key of the deploy telemetry value that
	// reports lifecycle.started for this type.
	LifecycleStartedTelemetryKey() string
}
