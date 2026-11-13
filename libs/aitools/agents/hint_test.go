package agents

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

// claudeCtx returns a context with a captured, non-TTY stderr buffer (what a
// coding agent's shell tool gives the CLI) and an isolated HOME, cache dir and
// working directory, so detection and the session throttle read only the
// test's temp state. home is returned so the test can drop fixtures under it.
func claudeCtx(t *testing.T, home string) (context.Context, *bytes.Buffer) {
	t.Chdir(t.TempDir())
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
	return telemetry.WithNewLogger(ctx), stderr
}

// commandAt builds the command tree for a space-separated path under a
// "databricks" root and returns the leaf, so ancestry checks see the same
// shape as the real CLI.
func commandAt(path string) *cobra.Command {
	cmd := &cobra.Command{Use: "databricks"}
	for name := range strings.FieldsSeq(path) {
		child := &cobra.Command{Use: name}
		cmd.AddCommand(child)
		cmd = child
	}
	return cmd
}

func airCmd() *cobra.Command { return commandAt("air list") }

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

// assertHinted checks for exactly one hint line and one hint-shown event of
// kind INSTALL_TEXT (T8).
func assertHinted(t *testing.T, ctx context.Context, stderr *bytes.Buffer) {
	t.Helper()
	assert.Equal(t, airHintText+"\n", stderr.String())
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx))
}

// assertSilent checks for no hint line and no telemetry event (T8).
func assertSilent(t *testing.T, ctx context.Context, stderr *bytes.Buffer) {
	t.Helper()
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestAIRHintText(t *testing.T) {
	// The hint names the skill, says what it is for, and offers both the install
	// and a no-install preview, on one line.
	assert.Contains(t, airHintText, "databricks-ai-runtime skill is not installed")
	assert.Contains(t, airHintText, "workload YAML")
	assert.Contains(t, airHintText, "databricks aitools install --skills-only --skills databricks-ai-runtime")
	assert.Contains(t, airHintText, "databricks aitools install --path <dir> --skills databricks-ai-runtime")
	assert.NotContains(t, airHintText, "\n")
}

func TestMaybeHint_HumanCallerSilent(t *testing.T) {
	// No agent env: AgentProvider() == "", so no hint even with nothing installed.
	testutil.SetAgentEnv(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	MaybeHint(ctx, airCmd())
	assertSilent(t, ctx, stderr)
}

// T1: every air subcommand, at any depth, gets the skill-specific hint.
func TestMaybeHint_AIRSubcommandsHint(t *testing.T) {
	setClaudeAgent(t)
	for _, path := range []string{
		"air run", "air get", "air list", "air logs", "air cancel",
		"air convert-to-dabs", "air images", "air images push", "air pools", "air pools list",
	} {
		t.Run(path, func(t *testing.T) {
			ctx, stderr := claudeCtx(t, t.TempDir())
			MaybeHint(ctx, commandAt(path))
			assertHinted(t, ctx, stderr)
		})
	}
}

// T3: commands outside air are never hinted, including the auth and version
// commands the generic hint used to fire on.
func TestMaybeHint_NonAIRCommandsSilent(t *testing.T) {
	setClaudeAgent(t)
	for _, path := range []string{
		"auth profiles", "current-user me", "clusters list", "jobs list", "bundle deploy",
		"workspace list", "version", "aitools install", "experimental aitools install",
		// The bare air node only renders help; its footer carries the tip.
		"air",
		// The hidden alias that points users at `databricks air`.
		"experimental air",
	} {
		t.Run(path, func(t *testing.T) {
			ctx, stderr := claudeCtx(t, t.TempDir())
			MaybeHint(ctx, commandAt(path))
			assertSilent(t, ctx, stderr)
		})
	}
}

func TestMaybeHint_PluginInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	// A plugin install writes only Claude's manifest, no skills subdir, and
	// bundles every Databricks skill including databricks-ai-runtime.
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)

	MaybeHint(ctx, airCmd())
	assertSilent(t, ctx, stderr)
}

func TestAIRSkillAvailable_LogsPluginDetection(t *testing.T) {
	t.Chdir(t.TempDir())
	home := t.TempDir()
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.6"}]}}`)

	// Capture debug output by installing a text logger on the context.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: log.LevelDebug})
	ctx := log.NewContext(t.Context(), slog.New(handler))
	ctx = env.WithUserHomeDir(ctx, home)

	assert.True(t, airSkillAvailable(ctx))
	assert.Contains(t, buf.String(), "Databricks plugin active for Claude Code")
	assert.Contains(t, buf.String(), "skipping AI Runtime skill hint")
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

			MaybeHint(ctx, airCmd())
			if tc.wantHint {
				assertHinted(t, ctx, stderr)
			} else {
				assertSilent(t, ctx, stderr)
			}
		})
	}
}

// T4: only the databricks-ai-runtime skill itself, in a location Claude loads
// for this working directory, silences the hint.
func TestMaybeHint_AIRSkillDetection(t *testing.T) {
	setClaudeAgent(t)
	for _, tc := range []struct {
		name     string
		setup    func(t *testing.T, home, cwd string)
		wantHint bool
	}{
		{"nothing installed", func(t *testing.T, home, cwd string) {}, true},
		{"canonical store", func(t *testing.T, home, cwd string) {
			writeCanonicalSkill(t, home, airSkillName)
		}, false},
		{"legacy store", func(t *testing.T, home, cwd string) {
			require.NoError(t, os.MkdirAll(filepath.Join(home, LegacySkillsDir, airSkillName), 0o755))
		}, false},
		{"canonical store via symlink", func(t *testing.T, home, cwd string) {
			dir := filepath.Join(home, CanonicalSkillsDir)
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(dir, airSkillName)))
		}, false},
		{"project store in cwd", func(t *testing.T, home, cwd string) {
			require.NoError(t, os.MkdirAll(filepath.Join(cwd, CanonicalSkillsDir, airSkillName), 0o755))
		}, false},
		{"other databricks skills only", func(t *testing.T, home, cwd string) {
			writeCanonicalSkill(t, home, "databricks")
			writeCanonicalSkill(t, home, "databricks-jobs")
			require.NoError(t, os.MkdirAll(filepath.Join(cwd, CanonicalSkillsDir, "databricks-apps"), 0o755))
		}, true},
		{"a file named like the skill", func(t *testing.T, home, cwd string) {
			dir := filepath.Join(home, CanonicalSkillsDir)
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, airSkillName), nil, 0o600))
		}, true},
		{"plugin for this project", func(t *testing.T, home, cwd string) {
			writePluginManifest(t, home, `{"plugins":{"databricks@m":[{"scope":"project","projectPath":"`+cwd+`"}]}}`)
		}, false},
		{"plugin for an unrelated project", func(t *testing.T, home, cwd string) {
			writePluginManifest(t, home, `{"plugins":{"databricks@m":[{"scope":"project","projectPath":"`+t.TempDir()+`"}]}}`)
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			cwd, err := os.Getwd()
			require.NoError(t, err)
			tc.setup(t, home, cwd)

			MaybeHint(ctx, airCmd())
			if tc.wantHint {
				assertHinted(t, ctx, stderr)
			} else {
				assertSilent(t, ctx, stderr)
			}
		})
	}
}

// T4: a project-scoped skill at the repository root still counts when the CLI
// runs in a subdirectory, as Claude Code loads it there.
func TestMaybeHint_AIRSkillInRepositoryRoot(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	root, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, CanonicalSkillsDir, airSkillName), 0o755))
	sub := filepath.Join(root, "training", "configs")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	t.Chdir(sub)

	MaybeHint(ctx, airCmd())
	assertSilent(t, ctx, stderr)
}

func TestMaybeHint_NoIONoPanic(t *testing.T) {
	setClaudeAgent(t)
	// Bare context without cmdio installed: cobra resolves --help and bare
	// invocations before PersistentPreRunE runs, so HasIO can be false here.
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	assert.NotPanics(t, func() { MaybeHint(ctx, airCmd()) })
	assert.Empty(t, hintKinds(t, ctx))
}

// T5 and T6: with a session id the first air command in the session is
// hinted and later ones stay quiet until the marker expires; without one
// (older Claude Code) every air command is hinted. Both cases run with a
// non-TTY stderr, which is how a coding agent's shell tool runs the CLI.
func TestMaybeHint_SessionThrottle(t *testing.T) {
	setClaudeAgent(t)
	for _, tc := range []struct {
		name       string
		session    string
		wantSecond bool
	}{
		{"session id: first air command only", "session-abc", false},
		{"no session id: every air command", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			cacheDir := t.TempDir()
			newCtx := func() (context.Context, *bytes.Buffer) {
				ctx, stderr := claudeCtx(t, home)
				ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
				if tc.session != "" {
					ctx = env.Set(ctx, claudeSessionEnv, tc.session)
				}
				return ctx, stderr
			}

			// A non-air command first must not use up the session's hint.
			ctx0, stderr0 := newCtx()
			MaybeHint(ctx0, commandAt("auth profiles"))
			assertSilent(t, ctx0, stderr0)

			ctx1, stderr1 := newCtx()
			MaybeHint(ctx1, commandAt("air list"))
			assertHinted(t, ctx1, stderr1)

			ctx2, stderr2 := newCtx()
			MaybeHint(ctx2, commandAt("air run"))
			if tc.wantSecond {
				assertHinted(t, ctx2, stderr2)
			} else {
				assertSilent(t, ctx2, stderr2)
			}
		})
	}
}

func TestMaybeHint_OtherSessionStillHinted(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	cacheDir := t.TempDir()
	for _, session := range []string{"session-a", "session-b"} {
		ctx, stderr := claudeCtx(t, home)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
		ctx = env.Set(ctx, claudeSessionEnv, session)
		MaybeHint(ctx, airCmd())
		assertHinted(t, ctx, stderr)
	}
}
