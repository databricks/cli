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
)

// Enabled reports whether the skills feature is on. It is the gate for the
// whole feature: unless DATABRICKS_SKILLS_PREVIEW is set to a true value, the
// skills command group is not registered.
func Enabled(ctx context.Context) bool {
	return envTrue(ctx, previewEnv)
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
