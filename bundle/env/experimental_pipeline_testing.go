package env

import (
	"context"

	envlib "github.com/databricks/cli/libs/env"
)

// ExperimentalPipelineTestingVariable enables the experimental testing property of pipelines.
const ExperimentalPipelineTestingVariable = "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_PIPELINE_TESTING"

// ExperimentalPipelineTesting reports whether the experimental testing property of pipelines is
// enabled. While it is off, bundles deploy pipelines exactly as they did before the property existed.
func ExperimentalPipelineTesting(ctx context.Context) bool {
	enabled, _ := envlib.GetBool(ctx, ExperimentalPipelineTestingVariable)
	return enabled
}
