package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// The build phase builds artifacts.
func Build(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	log.Info(ctx, "Phase: build")

	return bundle.ApplySeq(ctx, b,
		scripts.Execute(config.ScriptPreBuild),
		whl.DetectPackage(),
		artifacts.InferMissingProperties(),
		artifacts.PrepareAll(),
		artifacts.BuildAll(),
		scripts.Execute(config.ScriptPostBuild),
		mutator.ResolveVariableReferences(
			"artifacts",
		),
	)
}
