package resources

import (
	"reflect"

	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/jsonschema"
)

// SchemaLaunchStageDeclarer is implemented by a resource type that the bundle
// JSON schema publishes at a fixed launch stage. The stage applies to the type
// and to each of its fields, unless the API contract marks a field as more
// restrictive. Declaring it decouples the resource's label from its API's own
// launch stage, so an upstream stage change cannot silently alter it.
type SchemaLaunchStageDeclarer interface {
	SchemaLaunchStage() clijson.LaunchStage
}

// SchemaTransformer is implemented by a resource type whose bundle JSON schema
// differs from what reflection on its Go types produces, for example to hide
// fields that only the CLI sets.
type SchemaTransformer interface {
	// SchemaTransforms maps a Go type to a function that adjusts its schema.
	// The keys are the resource type itself or SDK types that it owns.
	SchemaTransforms() map[reflect.Type]func(jsonschema.Schema) jsonschema.Schema
}
