package skills

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

const (
	// previewEnv turns the skills feature on. Off, the skills command group is
	// not registered and nothing else in the feature runs.
	previewEnv = "DATABRICKS_SKILLS_PREVIEW"

	// hintsEnv turns on skill hints. It only has an effect while previewEnv is set.
	hintsEnv = "DATABRICKS_SKILLS_HINTS"

	// agentFeaturesDisabledEnv turns the agent features off: the skills feature
	// and the aitools hints and auto-install. It wins over previewEnv and hintsEnv.
	agentFeaturesDisabledEnv = "DATABRICKS_AGENT_FEATURES_DISABLED"
)

// AgentFeaturesDisabled reports whether DATABRICKS_AGENT_FEATURES_DISABLED is
// set to a true value. The aitools hints and auto-install check it too. The
// explicit `databricks aitools` commands still work.
func AgentFeaturesDisabled(ctx context.Context) bool {
	return envTrue(ctx, agentFeaturesDisabledEnv)
}

// Enabled reports whether the skills feature is on. It is the gate for the
// whole feature: unless DATABRICKS_SKILLS_PREVIEW is set to a true value, the
// skills command group is not registered. It is never on while
// DATABRICKS_AGENT_FEATURES_DISABLED is set.
func Enabled(ctx context.Context) bool {
	return !AgentFeaturesDisabled(ctx) && envTrue(ctx, previewEnv)
}

// HintsEnabled reports whether skill hints are on. Hints need the feature on
// and DATABRICKS_SKILLS_HINTS set to a true value of their own.
func HintsEnabled(ctx context.Context) bool {
	return Enabled(ctx) && envTrue(ctx, hintsEnv)
}

func envTrue(ctx context.Context, key string) bool {
	on, _ := env.GetBool(ctx, key)
	return on
}
