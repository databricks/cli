package generate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// loadStateForGenerate initializes the bundle, pulls the deployment state, tags the
// resolved engine in the user agent, and loads the state into the bundle. Shared by the
// resource-key paths of "generate dashboard" and "generate genie-space" - the only
// generate commands that resolve a bundle resource against deployment state. Errors are
// reported through logdiag; callers check logdiag.HasError on the returned context.
func loadStateForGenerate(ctx context.Context, b *bundle.Bundle) context.Context {
	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return ctx
	}

	requiredEngine, err := utils.ResolveEngineSetting(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return ctx
	}

	stateDesc := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true), requiredEngine)
	if logdiag.HasError(ctx) {
		return ctx
	}
	ctx = useragent.InContext(ctx, "engine", string(stateDesc.Engine))

	var state statemgmt.ExportedResourcesMap
	if stateDesc.Engine.IsDirect() {
		if err := utils.OpenDirectStateForRead(ctx, b, stateDesc); err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
		state = b.DeploymentBundle.ExportState(ctx)
	} else {
		state, err = terraform.ParseResourcesState(ctx, b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
	}

	bundle.ApplySeqContext(ctx, b, statemgmt.Load(state))
	return ctx
}
