package skills

import (
	"context"
	"sync"

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

	// AgentFeaturesDisabledEnv turns the agent features off: the skills feature
	// and the aitools hints and auto-install. It wins over previewEnv and hintsEnv.
	AgentFeaturesDisabledEnv = "DATABRICKS_AGENT_FEATURES_DISABLED"
)

// hintsMemoKey is the context key of the *hintsMemo that WithHintsMemo sets.
type hintsMemoKey struct{}

// hintsMemo holds the hints value resolved for one command run.
type hintsMemo struct {
	mu       sync.Mutex
	resolved bool
	on       bool
}

// AgentFeaturesDisabled reports whether DATABRICKS_AGENT_FEATURES_DISABLED is
// set to a true value. The aitools hints and auto-install check it too. The
// explicit `databricks aitools` commands still work.
func AgentFeaturesDisabled(ctx context.Context) bool {
	return envTrue(ctx, AgentFeaturesDisabledEnv)
}

// Enabled reports whether the skills feature is on. It is the gate for the
// whole feature: unless DATABRICKS_SKILLS_PREVIEW is set to a true value, the
// skills command group is not registered. It is never on while
// DATABRICKS_AGENT_FEATURES_DISABLED is set.
func Enabled(ctx context.Context) bool {
	return !AgentFeaturesDisabled(ctx) && envTrue(ctx, previewEnv)
}

// WithHintsMemo returns ctx with the hints value resolved once for the run.
// Every HintsEnabled call on the returned context, and on contexts derived from
// it, gives the answer of the first call. A cache refresh later in the run,
// such as the auto-install before the deferred hint prints, cannot then change
// that hint's text, its throttle or the skills command exemption. The memo is
// safe for concurrent use.
func WithHintsMemo(ctx context.Context) context.Context {
	return context.WithValue(ctx, hintsMemoKey{}, &hintsMemo{})
}

// HintsEnabled reports whether skill hints are on. Hints need the feature on.
// Then DATABRICKS_SKILLS_HINTS, when set, decides in both directions; otherwise
// the skills_hints entry of the compatibility manifest does. It never fetches;
// see internal/build/README.md. On a context from WithHintsMemo it answers once
// per run; otherwise it reads the manifest cache on every call.
func HintsEnabled(ctx context.Context) bool {
	if AgentFeaturesDisabled(ctx) {
		return false
	}
	if !Enabled(ctx) {
		return false
	}
	memo, ok := ctx.Value(hintsMemoKey{}).(*hintsMemo)
	if !ok {
		return resolveHints(ctx)
	}
	memo.mu.Lock()
	defer memo.mu.Unlock()
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
