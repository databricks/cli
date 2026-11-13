package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hintEnvVars are the non-detection env vars MaybeHint reads. They are cleared
// so a test run from inside Claude Code or CI sees none of the host's values.
var hintEnvVars = []string{
	claudeChildSessionEnv, claudeSessionEnv, ciEnv, "CLAUDE_CONFIG_DIR",
	// CI/CD provider markers read by useragent.CiCdProvider().
	"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TF_BUILD", "CIRCLECI", "TRAVIS",
	"BITBUCKET_BUILD_NUMBER", "PROJECT_ID", "CODEBUILD_BUILD_ARN", "TFC_RUN_ID",
}

// unsetEnv unsets vars for the test and restores them afterwards. t.Setenv
// cannot unset, hence the manual save/restore.
func unsetEnv(t *testing.T, vars []string) {
	for _, v := range vars {
		if orig, ok := os.LookupEnv(v); ok {
			t.Cleanup(func() { os.Setenv(v, orig) }) //nolint:usetesting // restoring, not setting fresh
			os.Unsetenv(v)                           //nolint:usetesting // t.Setenv cannot unset
		}
	}
}

// setClaudeAgent makes useragent.AgentProvider() report Claude Code for this
// test, with the host's agent and hint env cleared. These tests must not run
// in parallel (see testutil.SetAgentEnv). The first t.TempDir call runs before
// SetAgentEnv clears TMPDIR / TEMP, so later calls in the test still work.
// It also moves into an empty project with its own .git boundary, so project
// skills in the checkout running the tests are not detected.
func setClaudeAgent(t *testing.T) {
	cwd := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
	t.Chdir(cwd)
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
	unsetEnv(t, hintEnvVars)
}

// claudeCtx returns a context with a captured stderr buffer and an isolated
// HOME + cache dir, so detection and the session throttle read only the test's
// temp state. home is returned so the test can drop fixtures under it.
func claudeCtx(t *testing.T, home string) (context.Context, *bytes.Buffer) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
	// Throttle markers live in the file cache, which a host could disable.
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
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

// writeClaudeSkill puts a skill dir in Claude Code's own skills dir. It is a
// real dir rather than the installer's symlink so the fixture needs no symlink
// privilege on Windows; HasDatabricksSkillsIn treats both alike.
func writeClaudeSkill(t *testing.T, home, name string) {
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills", name), 0o755))
}

func TestMaybeHint_HumanCallerSilent(t *testing.T) {
	// No agent env: AgentProvider() == "", so no hint even with nothing installed.
	testutil.SetAgentEnv(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestMaybeHint_ClaudeNotInstalledHints(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
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

	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

// A project or local plugin only loads inside its own project, so elsewhere
// Claude Code still lacks the tooling and gets the hint.
func TestMaybeHint_ProjectPluginCountsOnlyInItsProject(t *testing.T) {
	project, other := t.TempDir(), t.TempDir()
	for _, tc := range []struct {
		name        string
		projectPath string
		wantHint    bool
	}{
		{name: "plugin for this project", projectPath: project},
		{name: "plugin for another project", projectPath: other, wantHint: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			manifest, err := json.Marshal(map[string]any{"plugins": map[string]any{
				"databricks@claude-plugins-official": []map[string]string{{"scope": "project", "projectPath": tc.projectPath}},
			}})
			require.NoError(t, err)
			writePluginManifest(t, home, string(manifest))
			t.Chdir(project)

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			if tc.wantHint {
				assert.Contains(t, stderr.String(), hintText)
			} else {
				assert.Empty(t, stderr.String())
			}
		})
	}
}

func TestClaudeToolingInstalled_LogsPluginVersion(t *testing.T) {
	unsetEnv(t, hintEnvVars)
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

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
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
	writeClaudeSkill(t, home, "databricks")

	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_SharedStoreOnlyStillHints(t *testing.T) {
	// Skills in the shared store (or the legacy store) but not in Claude Code's
	// own skills dir were installed for another agent; Claude doesn't load them.
	for _, dir := range []string{CanonicalSkillsDir, LegacySkillsDir} {
		t.Run(dir, func(t *testing.T) {
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			require.NoError(t, os.MkdirAll(filepath.Join(home, dir, "databricks"), 0o755))

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			assert.Equal(t, hintText+"\n", stderr.String())
		})
	}
}

func TestMaybeHint_NonDatabricksSkillStillHints(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeCanonicalSkill(t, home, "some-other-skill")

	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Contains(t, stderr.String(), hintText)
}

func TestMaybeHint_AIToolsCommandSilent(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)

	parent := &cobra.Command{Use: aiToolsCommandName}
	child := &cobra.Command{Use: "install"}
	parent.AddCommand(child)

	MaybeHint(ctx, child, noPluginRecorded)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestMaybeHint_NoIONoPanic(t *testing.T) {
	setClaudeAgent(t)
	// Bare context without cmdio installed: cobra resolves --help and bare
	// invocations before PersistentPreRunE runs, so HasIO can be false here.
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	assert.NotPanics(t, func() { MaybeHint(ctx, regularCmd(), noPluginRecorded) })
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
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Contains(t, stderr1.String(), hintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx1))

	// Same session id, second command: must stay quiet and log nothing.
	ctx2, stderr2 := newCtx()
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())
	assert.Empty(t, hintKinds(t, ctx2))
}

// wantNativeHint is the exact line Claude Code parses; see
// https://code.claude.com/docs/en/plugins/cli-hints. Spelled out rather than built
// from constants so a change to the plugin id or marketplace fails here.
const wantNativeHint = `<claude-code-hint v="1" type="plugin" value="databricks@claude-plugins-official" />`

func TestClaudeNativeHint_Format(t *testing.T) {
	assert.Equal(t, wantNativeHint, claudeNativeHint)
}

// authDockerCmd returns `auth docker <use>` as a leaf command.
func authDockerCmd(use string) *cobra.Command {
	auth := &cobra.Command{Use: authCommandName}
	docker := &cobra.Command{Use: "docker"}
	leaf := &cobra.Command{Use: use}
	auth.AddCommand(docker)
	docker.AddCommand(leaf)
	return leaf
}

func TestSelfExecEnv(t *testing.T) {
	assert.Equal(t, []string{"A=1", selfExecEnv + "=1"}, SelfExecEnv([]string{"A=1"}))
}

func TestMaybeHint_NativeHint(t *testing.T) {
	for _, tc := range []struct {
		name       string
		env        map[string]string
		setup      func(t *testing.T, home string)
		cmd        func() *cobra.Command
		wantStderr string
	}{
		{
			name:       "claude child session prints native then text",
			env:        map[string]string{claudeChildSessionEnv: "1"},
			wantStderr: wantNativeHint + "\n" + hintText + "\n",
		},
		{
			name:       "no child session prints text only",
			wantStderr: hintText + "\n",
		},
		{
			name:       "CI prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "true"},
			wantStderr: hintText + "\n",
		},
		{
			name:       "CI=false still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "false"},
			wantStderr: wantNativeHint + "\n" + hintText + "\n",
		},
		{
			// Some providers set CI to their own name.
			name:       "CI set to a provider name prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "woodpecker"},
			wantStderr: hintText + "\n",
		},
		{
			// Azure Pipelines sets TF_BUILD but not CI.
			name:       "CI provider without CI var prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", "TF_BUILD": "True"},
			wantStderr: hintText + "\n",
		},
		{
			name: "plugin installed prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			setup: func(t *testing.T, home string) {
				writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)
			},
		},
		{
			name: "skills installed prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			setup: func(t *testing.T, home string) {
				writeClaudeSkill(t, home, "databricks")
			},
		},
		{
			name: "auth token prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd: func() *cobra.Command {
				parent := &cobra.Command{Use: authCommandName}
				child := &cobra.Command{Use: "token [PROFILE]"}
				parent.AddCommand(child)
				return child
			},
		},
		{
			name:       "other auth command still hints",
			env:        map[string]string{claudeChildSessionEnv: "1"},
			wantStderr: wantNativeHint + "\n" + hintText + "\n",
			cmd: func() *cobra.Command {
				parent := &cobra.Command{Use: authCommandName}
				child := &cobra.Command{Use: "profiles"}
				parent.AddCommand(child)
				return child
			},
		},
		{
			name: "auth docker token prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd:  func() *cobra.Command { return authDockerCmd("token") },
		},
		{
			name:       "other auth docker command still hints",
			env:        map[string]string{claudeChildSessionEnv: "1"},
			wantStderr: wantNativeHint + "\n" + hintText + "\n",
			cmd:        func() *cobra.Command { return authDockerCmd("configure [PROFILE]") },
		},
		{
			// air images push runs `auth docker configure` and embeds its
			// output in an error; the child must add nothing to it.
			name: "CLI self-exec child prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1", selfExecEnv: "1"},
		},
		{
			name: "aitools command prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd: func() *cobra.Command {
				parent := &cobra.Command{Use: aiToolsCommandName}
				child := &cobra.Command{Use: "install"}
				parent.AddCommand(child)
				return child
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			// CiCdProvider caches its first answer process-wide.
			useragent.ClearCache()
			if tc.setup != nil {
				tc.setup(t, home)
			}
			cmd := regularCmd()
			if tc.cmd != nil {
				cmd = tc.cmd()
			}

			MaybeHint(ctx, cmd, noPluginRecorded)
			assert.Equal(t, tc.wantStderr, stderr.String())
		})
	}
}

func TestMaybeHint_NativeHintSilentForOtherCallers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		agentEnv []string
		wantText string
	}{
		{"human", nil, ""},
		// Other agents get only their own text hint.
		{"other agent", []string{"CURSOR_AGENT", "1"}, agentInstallHint(ByName(NameCursor), noPluginRecorded).text + "\n"},
		{"multiple agents", []string{"CLAUDECODE", "1", "CURSOR_AGENT", "1"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetAgentEnv(t, tc.agentEnv...)
			unsetEnv(t, hintEnvVars)
			ctx, stderr := claudeCtx(t, t.TempDir())
			// Even with Claude's child-session marker present, only a sole
			// Claude Code caller gets the native hint.
			ctx = env.Set(ctx, claudeChildSessionEnv, "1")

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			assert.Equal(t, tc.wantText, stderr.String())
		})
	}
}

func TestMaybeHint_NativeHintThrottledPerSession(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	cacheDir := t.TempDir()
	newCtx := func() (context.Context, *bytes.Buffer) {
		ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
		ctx = env.WithUserHomeDir(ctx, home)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
		ctx = env.Set(ctx, claudeSessionEnv, "session-abc")
		ctx = env.Set(ctx, claudeChildSessionEnv, "1")
		return ctx, stderr
	}

	ctx1, stderr1 := newCtx()
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr1.String())

	// Same session, second command: both lines stay quiet, so the tag doesn't
	// reach a downstream reader such as `2>&1 | jq` on every command.
	ctx2, stderr2 := newCtx()
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())

	// Another session: both lines print again.
	ctx3, stderr3 := newCtx()
	ctx3 = env.Set(ctx3, claudeSessionEnv, "session-def")
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr3.String())
}

func TestMaybeHint_NativeHintSilentOnTerminal(t *testing.T) {
	// The fakes make one stream look like a terminal, as in tmux or screen
	// started from inside Claude Code, where nothing strips the tag. A person
	// typing there has the terminal on stdin even when the output is piped, as
	// in `databricks ... 2>&1 | less`.
	for _, tc := range []struct {
		name                  string
		stdin, stdout, stderr bool
	}{
		{name: "stdin", stdin: true},
		{name: "stdout", stdout: true},
		{name: "stderr", stderr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setClaudeAgent(t)
			stderr := &bytes.Buffer{}
			var in io.Reader
			out := io.Discard
			var errOut io.Writer = stderr
			if tc.stdin {
				in = cmdio.FakeTTYReader(&bytes.Buffer{})
			}
			if tc.stdout {
				out = cmdio.FakeTTY(out)
			}
			if tc.stderr {
				errOut = cmdio.FakeTTY(errOut)
			}
			ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, in, out, errOut, "", ""))
			ctx = env.WithUserHomeDir(ctx, t.TempDir())
			ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
			ctx = env.Set(ctx, claudeChildSessionEnv, "1")

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			assert.Equal(t, hintText+"\n", stderr.String())
		})
	}
}

func TestAgentHint_NativeLineNilIsEmpty(t *testing.T) {
	h := &agentHint{}
	assert.Empty(t, h.nativeLine(t.Context()))
}

func TestHintForAgent(t *testing.T) {
	claude := hintForAgent(claudeAgentProvider, noPluginRecorded)
	require.NotNil(t, claude)
	assert.Equal(t, hintText, claude.text)
	assert.Equal(t, claudeSessionEnv, claude.sessionEnv)

	assert.Nil(t, hintForAgent("", noPluginRecorded))
	assert.Nil(t, hintForAgent("multiple", noPluginRecorded))
	assert.Nil(t, hintForAgent("windsurf", noPluginRecorded))

	cursor := hintForAgent("cursor", noPluginRecorded)
	require.NotNil(t, cursor)
	assert.Equal(t, protos.AitoolsHintKindAgentInstallText, cursor.textKind)
	assert.Nil(t, cursor.native)
}

func TestMaybeHint_LogsEventsOncePerSession(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	cacheDir := t.TempDir()
	newCtx := func(childSession bool) (context.Context, *bytes.Buffer) {
		ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
		ctx = telemetry.WithNewLogger(ctx)
		ctx = env.WithUserHomeDir(ctx, home)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
		ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
		ctx = env.Set(ctx, claudeSessionEnv, "session-abc")
		if childSession {
			ctx = env.Set(ctx, claudeChildSessionEnv, "1")
		}
		return ctx, stderr
	}

	ctx1, _ := newCtx(true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx1))
	events := telemetry.BufferedEvents(ctx1)
	require.Len(t, events, 2)

	// Throttled for the session: nothing prints and nothing logs, so the
	// command doesn't pay for a telemetry upload at exit.
	ctx2, stderr2 := newCtx(true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx2))

	// Throttled and not a Claude Code subprocess: nothing prints, nothing logs.
	ctx3, stderr3 := newCtx(false)
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr3.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx3))
}

func TestMaybeHint_InstalledLogsNothing(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, _ := claudeCtx(t, home)
	ctx = telemetry.WithNewLogger(ctx)
	ctx = env.Set(ctx, claudeChildSessionEnv, "1")
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)

	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, telemetry.BufferedEvents(ctx))
}
