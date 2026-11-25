package metrics

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// DeployBoolValues holds, per resource type, a function that returns the boolean values
// the type reports about the bundle's resources in deploy telemetry. The file of each
// type in this package registers its function from init(). LogDeployTelemetry in
// bundle/phases runs them and reports the values type by type in the canonical order of
// resource types, and within a type in the order its function returns them.
var DeployBoolValues = config.NewResourceTypeRegistry[func(*bundle.Bundle) []protos.BoolMapEntry]("deploy telemetry bool values")
