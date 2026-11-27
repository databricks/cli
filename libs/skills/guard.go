package skills

import (
	"context"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

const (
	// previewEnv turns the skills feature on. Off, the skills command group is
	// not registered and nothing else in the feature runs.
	previewEnv = "DATABRICKS_SKILLS_PREVIEW"

	// hintsEnv forces skill hints on or off. It only has an effect while
	// previewEnv is set. Unset, the compatibility manifest decides.
	hintsEnv = "DATABRICKS_SKILLS_HINTS"
)

// hintsMemoKey is the context key of the *hintsMemo that WithHintsMemo sets.
type hintsMemoKey struct{}

// hintsMemo holds the hints value resolved for one command run.
type hintsMemo struct {
	resolved bool
	on       bool
}

// Enabled reports whether the skills feature is on. It is the gate for the
// whole feature: unless DATABRICKS_SKILLS_PREVIEW is set to a true value, the
// skills command group is not registered.
func Enabled(ctx context.Context) bool {
	return envTrue(ctx, previewEnv)
}

// WithHintsMemo returns ctx with the hints value resolved once for the run.
// Every HintsEnabled call on the returned context, and on contexts derived from
// it, gives the answer of the first call. A cache refresh later in the run,
// such as the auto-install before the deferred hint prints, cannot then change
// that hint's text, its throttle or the skills command exemption.
func WithHintsMemo(ctx context.Context) context.Context {
	return context.WithValue(ctx, hintsMemoKey{}, &hintsMemo{})
}

// HintsEnabled reports whether skill hints are on. Hints need the feature on.
// Then DATABRICKS_SKILLS_HINTS, when set, decides in both directions; otherwise
// the skills_hints entry of the compatibility manifest does. It never fetches;
// see internal/build/README.md. On a context from WithHintsMemo it answers once
// per run; otherwise it reads the manifest cache on every call.
func HintsEnabled(ctx context.Context) bool {
	if !Enabled(ctx) {
		return false
	}
	memo, ok := ctx.Value(hintsMemoKey{}).(*hintsMemo)
	if !ok {
		return resolveHints(ctx)
	}
	if !memo.resolved {
		memo.on = resolveHints(ctx)
		memo.resolved = true
	}
	return memo.on
}

// resolveHints reads the hints value with the feature on: the environment
// switch when set, else the manifest entry for this CLI version.
func resolveHints(ctx context.Context) bool {
	if on, ok := env.GetBool(ctx, hintsEnv); ok {
		return on
	}
	on, err := clicompat.ResolveSkillsHints(ctx, build.GetInfo().Version)
	if err != nil {
		log.Debugf(ctx, "Skill hints stay off: %v", err)
		return false
	}
	return on
}

func envTrue(ctx context.Context, key string) bool {
	on, _ := env.GetBool(ctx, key)
	return on
}
