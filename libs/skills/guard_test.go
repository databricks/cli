package skills

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each case clears the process environment with testutil.CleanupEnvironment,
// so an exported value on the developer's machine does not leak in.
func TestEnabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset", want: false},
		{name: "true", set: true, value: "true", want: true},
		{name: "one", set: true, value: "1", want: true},
		{name: "false", set: true, value: "false", want: false},
		{name: "zero", set: true, value: "0", want: false},
		{name: "empty", set: true, value: "", want: false},
		{name: "invalid", set: true, value: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			ctx := t.Context()
			if c.set {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.value)
			}
			assert.Equal(t, c.want, Enabled(ctx))
		})
	}
}

// Only a true value turns DATABRICKS_AGENT_FEATURES_DISABLED on. Unset, false
// and an invalid value leave the feature as its own switches set it.
func TestAgentFeaturesDisabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "true", value: "true", want: true},
		{name: "one", value: "1", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "false", value: "false", want: false},
		{name: "zero", value: "0", want: false},
		{name: "invalid", value: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, c.value)
			assert.Equal(t, c.want, AgentFeaturesDisabled(ctx))
		})
	}
}

// With both skills switches on, a true DATABRICKS_AGENT_FEATURES_DISABLED turns
// Enabled and HintsEnabled off. Any other value leaves them on.
func TestAgentFeaturesDisabledOverridesSkillsSwitches(t *testing.T) {
	cases := []struct {
		name     string
		disabled string
		want     bool
	}{
		{name: "unset", want: true},
		{name: "true", disabled: "true", want: false},
		{name: "one", disabled: "1", want: false},
		{name: "false", disabled: "false", want: true},
		{name: "invalid", disabled: "maybe", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, c.disabled)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
			assert.Equal(t, c.want, Enabled(ctx))
			assert.Equal(t, c.want, HintsEnabled(ctx))
		})
	}
}

func TestHintsEnabled(t *testing.T) {
	// One entry applies to every CLI version, so the test does not depend on
	// which version the test binary reports.
	const entryOn = `{"0.1.0": {"appkit": "0.84.0", "skills": "0.2.30", "skills_hints": true}}`
	const entryOff = `{"0.1.0": {"appkit": "0.84.0", "skills": "0.2.30"}}`

	cases := []struct {
		name     string
		preview  string
		hints    string
		hintsSet bool
		manifest string // written to the local cache; empty means no cache file
		stale    bool
		want     bool
	}{
		{name: "both unset", want: false},
		{name: "preview off, hints on", preview: "false", hints: "true", hintsSet: true, want: false},
		{name: "preview unset, hints on", hints: "true", hintsSet: true, want: false},
		{name: "preview off, entry on", manifest: entryOn, want: false},
		{name: "preview off, hints on, entry on", hints: "true", hintsSet: true, manifest: entryOn, want: false},
		{name: "preview on, hints unset", preview: "true", want: false},
		{name: "preview on, hints false", preview: "true", hints: "false", hintsSet: true, want: false},
		{name: "preview on, hints invalid", preview: "true", hints: "maybe", hintsSet: true, want: false},
		{name: "preview on, hints empty, entry on", preview: "true", hints: "", hintsSet: true, manifest: entryOn, want: false},
		{name: "preview on, hints true", preview: "true", hints: "true", hintsSet: true, want: true},
		{name: "preview on, hints one", preview: "1", hints: "1", hintsSet: true, want: true},
		{name: "preview on, no cache", preview: "true", want: false},
		{name: "preview on, entry on", preview: "true", manifest: entryOn, want: true},
		{name: "preview on, entry on, stale cache ignored", preview: "true", manifest: entryOn, stale: true, want: false},
		{name: "preview on, entry without the field", preview: "true", manifest: entryOff, want: false},
		{name: "preview on, env false over entry on", preview: "true", hints: "false", hintsSet: true, manifest: entryOn, want: false},
		{name: "preview on, env invalid over entry on", preview: "true", hints: "maybe", hintsSet: true, manifest: entryOn, want: false},
		{name: "preview on, env true over entry without the field", preview: "true", hints: "true", hintsSet: true, manifest: entryOff, want: true},
		{name: "preview on, env true, no cache", preview: "true", hints: "true", hintsSet: true, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			ctx := t.Context()
			if c.preview != "" {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.preview)
			}
			if c.hintsSet {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)
			}
			if c.manifest != "" {
				modTime := time.Now()
				if c.stale {
					modTime = modTime.Add(-2 * time.Hour)
				}
				writeCachedManifest(t, ctx, c.manifest, modTime)
			}
			assert.Equal(t, c.want, HintsEnabled(ctx))
		})
	}
}

// writeCachedManifest writes manifest to the local manifest cache that
// libs/clicompat reads, with the given mod time.
func writeCachedManifest(t *testing.T, ctx context.Context, manifest string, modTime time.Time) {
	t.Helper()
	path := clicompat.LocalManifestPath(ctx)
	testutil.WriteFile(t, path, manifest)
	require.NoError(t, os.Chtimes(path, modTime, modTime))
}
