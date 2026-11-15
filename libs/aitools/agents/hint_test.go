package agents

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setClaudeAgent makes useragent.AgentProvider() report Claude Code for this
// test, with the host's environment cleared. These tests must not run in
// parallel (see testutil.SetAgentEnv).
func setClaudeAgent(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
}

// claudeCtx returns a context with a captured stderr buffer and an isolated
// HOME + cache dir, so detection and the session throttle read only the test's
// temp state. home is returned so the test can drop fixtures under it.
func claudeCtx(t *testing.T, home string) (context.Context, *bytes.Buffer) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
	return ctx, stderr
}

func regularCmd() *cobra.Command { return &cobra.Command{Use: "current-user"} }

func writePluginManifest(t *testing.T, home, body string) {
	dir := filepath.Join(home, ".claude", "plugins")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte(body), 0o600))
}

func writeCanonicalSkill(t *testing.T, home, name string) {
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".databricks", "aitools", "skills", name), 0o755))
}

// hintKinds returns the kinds of the hint-shown events buffered on ctx,
// failing on any other event so a stray event cannot hide behind the filter.
func hintKinds(t *testing.T, ctx context.Context) []protos.AitoolsHintKind {
	var out []protos.AitoolsHintKind
	for _, e := range telemetry.BufferedEvents(ctx) {
		require.NotNil(t, e.AitoolsHintEvent, "unexpected non-hint event: %+v", e)
		out = append(out, e.AitoolsHintEvent.HintKind)
	}
	return out
}

func TestMaybeHint_HumanCallerSilent(t *testing.T) {
	// No agent env: AgentProvider() == "", so no hint even with nothing installed.
	testutil.SetAgentEnv(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	MaybeHint(ctx, regularCmd())
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestMaybeHint_ClaudeNotInstalledHints(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	MaybeHint(ctx, regularCmd())
	assert.Contains(t, stderr.String(), hintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx))
}

func TestMaybeHint_PluginInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	// The core regression: a plugin install writes only Claude's manifest, no
	// skills subdir. Detection must treat it as installed.
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)
	ctx = telemetry.WithNewLogger(ctx)

	MaybeHint(ctx, regularCmd())
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestClaudeToolingInstalled_LogsPluginVersion(t *testing.T) {
	home := t.TempDir()
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.6"}]}}`)

	// Capture debug output by installing a text logger on the context.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: log.LevelDebug})
	ctx := log.NewContext(t.Context(), slog.New(handler))
	ctx = env.WithUserHomeDir(ctx, home)

	assert.True(t, claudeToolingInstalled(ctx))

	// The text handler escapes the inner quotes around the version, so assert on
	// the stable, unescaped fragments rather than the full quoted message.
	out := buf.String()
	assert.Contains(t, out, "Databricks plugin detected for Claude Code")
	assert.Contains(t, out, "0.2.6")
	assert.Contains(t, out, "skipping install hint")
}

func TestMaybeHint_CustomClaudeConfigDir(t *testing.T) {
	setClaudeAgent(t)
	for _, tc := range []struct {
		name             string
		customInstalled  bool
		defaultInstalled bool
		wantHint         bool
	}{
		{"custom plugin", true, false, false},
		{"both plugins", true, true, false},
		{"default plugin only", false, true, true},
		{"neither plugin", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			customHome := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			ctx = env.Set(ctx, "CLAUDE_CONFIG_DIR", filepath.Join(customHome, ".claude"))
			manifest := `{"plugins":{"databricks@local":[{"version":"1.0.0"}]}}`
			if tc.customInstalled {
				writePluginManifest(t, customHome, manifest)
			}
			if tc.defaultInstalled {
				writePluginManifest(t, home, manifest)
			}

			MaybeHint(ctx, regularCmd())
			if tc.wantHint {
				assert.Contains(t, stderr.String(), hintText)
			} else {
				assert.Empty(t, stderr.String())
			}
		})
	}
}

func TestMaybeHint_SkillsInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeCanonicalSkill(t, home, "databricks")

	MaybeHint(ctx, regularCmd())
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_LegacySkillsInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".databricks", "agent-skills", "databricks"), 0o755))

	MaybeHint(ctx, regularCmd())
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_NonDatabricksSkillStillHints(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeCanonicalSkill(t, home, "some-other-skill")

	MaybeHint(ctx, regularCmd())
	assert.Contains(t, stderr.String(), hintText)
}

func TestMaybeHint_AIToolsCommandSilent(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)

	parent := &cobra.Command{Use: aiToolsCommandName}
	child := &cobra.Command{Use: "install"}
	parent.AddCommand(child)

	MaybeHint(ctx, child)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestMaybeHint_NoIONoPanic(t *testing.T) {
	setClaudeAgent(t)
	// Bare context without cmdio installed: cobra resolves --help and bare
	// invocations before PersistentPreRunE runs, so HasIO can be false here.
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	assert.NotPanics(t, func() { MaybeHint(ctx, regularCmd()) })
	assert.Empty(t, hintKinds(t, ctx))
}

func TestMaybeHint_ThrottledPerSession(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	cacheDir := t.TempDir()
	newCtx := func() (context.Context, *bytes.Buffer) {
		ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
		ctx = env.WithUserHomeDir(ctx, home)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
		ctx = env.Set(ctx, claudeSessionEnv, "session-abc")
		return telemetry.WithNewLogger(ctx), stderr
	}

	ctx1, stderr1 := newCtx()
	MaybeHint(ctx1, regularCmd())
	assert.Contains(t, stderr1.String(), hintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx1))

	// Same session id, second command: must stay quiet and log nothing.
	ctx2, stderr2 := newCtx()
	MaybeHint(ctx2, regularCmd())
	assert.Empty(t, stderr2.String())
	assert.Empty(t, hintKinds(t, ctx2))
}
