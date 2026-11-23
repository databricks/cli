package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/aitools/autoinstall"
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

// setClaudeAgent makes useragent.AgentProvider() report Claude Code for this
// test, with the host's agent and hint env cleared. These tests must not run
// in parallel (see testutil.SetAgentEnv). It also moves into an empty project
// with its own .git boundary, so project skills in the checkout running the
// tests are not detected.
func setClaudeAgent(t *testing.T) {
	cwd := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
	t.Chdir(cwd)
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
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

func regularCmd() *cobra.Command { return commandAt("clusters list") }

func writePluginManifest(t *testing.T, home, body string) {
	dir := filepath.Join(home, ".claude", "plugins")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte(body), 0o600))
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

func TestClaudeToolingInstalled_LogsPluginDetected(t *testing.T) {
	// Clears the host's CLAUDE_CONFIG_DIR, which would move the manifest.
	testutil.SetAgentEnv(t)
	home := t.TempDir()
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.6"}]}}`)

	// Capture debug output by installing a text logger on the context.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: log.LevelDebug})
	ctx := log.NewContext(t.Context(), slog.New(handler))
	ctx = env.WithUserHomeDir(ctx, home)

	assert.True(t, claudeToolingInstalled(ctx))

	assert.Contains(t, buf.String(), "Databricks plugin detected for Claude Code; skipping install hint")
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
	writeClaudeSkill(t, home, "some-other-skill")

	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Contains(t, stderr.String(), hintText)
}

func TestMaybeHint_AIToolsCommandSilent(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)

	MaybeHint(ctx, commandAt("aitools install"), noPluginRecorded)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

// commandAt builds a command tree under a "databricks" root and returns the
// command at the space-separated path, e.g. "current-user me".
func commandAt(path string) *cobra.Command {
	cmd := &cobra.Command{Use: "databricks"}
	for name := range strings.FieldsSeq(path) {
		child := &cobra.Command{Use: name}
		cmd.AddCommand(child)
		cmd = child
	}
	return cmd
}

func TestIsHintExemptCommand(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"aitools install", true},
		{"experimental aitools skills install", true},
		{"feedback", true},
		{"auth login", true},
		{"auth token", true},
		{"configure", true},
		{"current-user me", true},
		{"completion bash", true},
		{"help", true},
		{cobra.ShellCompRequestCmd, true},
		{"version", true},
		{"authz", false},
		{"bundle help", false},
		{"current-user", false},
		{"clusters list", false},
		{"jobs run-now", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, isHintExemptCommand(commandAt(tc.path)))
		})
	}
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

// sessionCtx returns a context for one command in session-abc, sharing home
// and cacheDir with the session's other commands. childSession marks it as a
// Claude Code subprocess, which enables the native line.
func sessionCtx(t *testing.T, home, cacheDir string, childSession bool) (context.Context, *bytes.Buffer) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
	// Throttle markers live in the file cache, which a host could disable.
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, claudeSessionEnv, "session-abc")
	if childSession {
		ctx = env.Set(ctx, claudeChildSessionEnv, "1")
	}
	return ctx, stderr
}

func TestMaybeHint_ThrottledPerSession(t *testing.T) {
	setClaudeAgent(t)
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, stderr1 := sessionCtx(t, home, cacheDir, false)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Contains(t, stderr1.String(), hintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx1))

	// Same session id, second command: must stay quiet and log nothing.
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, false)
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
			name:       "CI=no still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "no"},
			wantStderr: wantNativeHint + "\n" + hintText + "\n",
		},
		{
			name:       "CI=OFF still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "OFF"},
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
		// The auth cases below root the tree at auth, so only the credential-helper
		// check applies; under the databricks root, hintExemptCommands covers all
		// of auth.
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
			cmd:  func() *cobra.Command { return commandAt("aitools install") },
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
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, stderr1 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr1.String())

	// Same session, second command: both lines stay quiet, so the tag doesn't
	// reach a downstream reader such as `2>&1 | jq` on every command.
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())

	// Another session: both lines print again.
	ctx3, stderr3 := sessionCtx(t, home, cacheDir, true)
	ctx3 = env.Set(ctx3, claudeSessionEnv, "session-def")
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr3.String())
}

// ageHintMarkers backdates every marker of throttle in cacheDir by age, as if
// it had been written that long ago.
func ageHintMarkers(t *testing.T, cacheDir string, throttle hintThrottle, age time.Duration) {
	markers, err := filepath.Glob(filepath.Join(cacheDir, "*", throttle.component, "*"))
	require.NoError(t, err)
	require.NotEmpty(t, markers)
	aged := time.Now().Add(-age)
	for _, m := range markers {
		require.NoError(t, os.Chtimes(m, aged, aged))
	}
}

// A native line withheld from a command (not a Claude Code subprocess, or in
// CI) must not burn the native throttle: the session's next Claude Code
// command still gets it, without the throttled text. That is the first native
// print in the text window, so it logs its event.
func TestMaybeHint_WithheldNativeLineKeepsItsThrottle(t *testing.T) {
	for _, tc := range []struct {
		name         string
		childSession bool
		ci           bool
	}{
		{name: "no child session"},
		{name: "CI", childSession: true, ci: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setClaudeAgent(t)
			// CiCdProvider caches its first answer process-wide.
			useragent.ClearCache()
			home, cacheDir := t.TempDir(), t.TempDir()

			ctx1, stderr1 := sessionCtx(t, home, cacheDir, tc.childSession)
			if tc.ci {
				ctx1 = env.Set(ctx1, ciEnv, "true")
			}
			MaybeHint(ctx1, regularCmd(), noPluginRecorded)
			assert.Equal(t, hintText+"\n", stderr1.String())
			assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx1))

			ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
			MaybeHint(ctx2, regularCmd(), noPluginRecorded)
			assert.Equal(t, wantNativeHint+"\n", stderr2.String())
			assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindClaudeCodeNative}, hintKinds(t, ctx2))
		})
	}
}

// The native line has its own, shorter throttle: once it expires the line
// prints again while the text hint stays throttled. Its event stays on the
// text's clock, so the repeat logs nothing and the command uploads no
// telemetry for it; a new text window logs both events again.
func TestMaybeHint_NativeOnlyRepeatLogsNoEvent(t *testing.T) {
	setClaudeAgent(t)
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, stderr1 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr1.String())
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx1))

	// Past the native TTL but not the text's: only the native line prints.
	ageHintMarkers(t, cacheDir, nativeHintThrottle, nativeHintThrottle.ttl+time.Minute)
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n", stderr2.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx2))

	// Both markers fresh again: nothing prints.
	ctx3, stderr3 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr3.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx3))

	// Past the text TTL: a new text window, so both lines and events are back.
	for _, throttle := range []hintThrottle{textHintThrottle, nativeHintThrottle, nativeEventThrottle} {
		ageHintMarkers(t, cacheDir, throttle, textHintThrottle.ttl+time.Minute)
	}
	ctx4, stderr4 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx4, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr4.String())
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx4))
}

// A text print without the native line starts a new text window, so the next
// native print logs its event even when a native event from the previous
// window is younger than the text TTL.
func TestMaybeHint_TextPrintStartsNewNativeEventWindow(t *testing.T) {
	setClaudeAgent(t)
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, _ := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	require.Len(t, hintKinds(t, ctx1), 2)

	// The text window expires, but not the native event marker; the next
	// command withholds the native line and prints only the text.
	ageHintMarkers(t, cacheDir, textHintThrottle, textHintThrottle.ttl+time.Minute)
	ageHintMarkers(t, cacheDir, nativeHintThrottle, nativeHintThrottle.ttl+time.Minute)
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, false)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Equal(t, hintText+"\n", stderr2.String())
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx2))

	ctx3, stderr3 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n", stderr3.String())
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindClaudeCodeNative}, hintKinds(t, ctx3))
}

// A command that prints both lines starts a new text window, so it logs the
// native event even when the previous window's first native line printed late
// and its event marker is still fresh.
func TestMaybeHint_TextAndNativePrintStartsNewNativeEventWindow(t *testing.T) {
	setClaudeAgent(t)
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, stderr1 := sessionCtx(t, home, cacheDir, false)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, hintText+"\n", stderr1.String())

	// Late in the text window, the first native line prints and logs its event.
	ageHintMarkers(t, cacheDir, textHintThrottle, 55*time.Minute)
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n", stderr2.String())
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindClaudeCodeNative}, hintKinds(t, ctx2))

	// The text and native markers expire; the event marker is still fresh.
	ageHintMarkers(t, cacheDir, textHintThrottle, textHintThrottle.ttl+time.Minute)
	ageHintMarkers(t, cacheDir, nativeHintThrottle, nativeHintThrottle.ttl+time.Minute)
	ctx3, stderr3 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx3, regularCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+hintText+"\n", stderr3.String())
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx3))
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
	home, cacheDir := t.TempDir(), t.TempDir()

	ctx1, _ := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx1))
	events := telemetry.BufferedEvents(ctx1)
	require.Len(t, events, 2)

	// Throttled for the session: nothing prints and nothing logs, so the
	// command doesn't pay for a telemetry upload at exit.
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx2))

	// Throttled and not a Claude Code subprocess: nothing prints, nothing logs.
	ctx3, stderr3 := sessionCtx(t, home, cacheDir, false)
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

// writeProjectSkill creates a project-scoped Claude skill symlink.
func writeProjectSkill(t *testing.T, project, name string) {
	canonical := filepath.Join(project, ".databricks", "aitools", "skills", name)
	require.NoError(t, os.MkdirAll(canonical, 0o755))
	link := filepath.Join(project, ".claude", "skills", name)
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	rel, err := filepath.Rel(filepath.Dir(link), canonical)
	require.NoError(t, err)
	require.NoError(t, os.Symlink(rel, link))
}

func TestMaybeHint_ProjectSkillsInstalledSilent(t *testing.T) {
	for _, tc := range []struct {
		name string
		// subdir is where the command runs, relative to the project root.
		subdir string
	}{
		{name: "project root"},
		{name: "repo subdirectory", subdir: "train"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			project := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(project, ".git"), 0o755))
			writeProjectSkill(t, project, "databricks-ai-runtime")
			cwd := filepath.Join(project, tc.subdir)
			require.NoError(t, os.MkdirAll(cwd, 0o755))
			t.Chdir(cwd)

			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			assert.Empty(t, stderr.String())
		})
	}
}

// setInstallsFor replaces autoinstall.InstallsFor for the test and returns the
// agent names it was asked about.
func setInstallsFor(t *testing.T, installs bool) *[]string {
	var asked []string
	prev := autoinstall.InstallsFor
	autoinstall.InstallsFor = func(_ *cobra.Command, agentName string) bool {
		asked = append(asked, agentName)
		return installs
	}
	t.Cleanup(func() { autoinstall.InstallsFor = prev })
	return &asked
}

func TestMaybeHint_WithheldWhileAutoInstallInstallsForTheAgent(t *testing.T) {
	setClaudeAgent(t)
	home, cacheDir := t.TempDir(), t.TempDir()
	asked := setInstallsFor(t, true)

	// Both lines are withheld, and no event or throttle marker is written.
	ctx1, stderr1 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx1, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr1.String())
	assert.Empty(t, hintKinds(t, ctx1))
	assert.Equal(t, []string{NameClaudeCode}, *asked)

	// That run did not install after all: the session's next command hints.
	setInstallsFor(t, false)
	ctx2, stderr2 := sessionCtx(t, home, cacheDir, true)
	MaybeHint(ctx2, regularCmd(), noPluginRecorded)
	assert.Contains(t, stderr2.String(), wantNativeHint)
	assert.Contains(t, stderr2.String(), hintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindClaudeCodeNative, protos.AitoolsHintKindInstallText}, hintKinds(t, ctx2))
}

func TestMaybeHint_RegistryAgentAskedByItsName(t *testing.T) {
	testutil.SetAgentEnv(t, "CURSOR_AGENT", "1")
	cwd := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
	t.Chdir(cwd)
	asked := setInstallsFor(t, true)
	ctx, stderr := claudeCtx(t, t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
	assert.Equal(t, []string{NameCursor}, *asked)
}

func TestMaybeHint_InstalledToolingSkipsTheAutoInstallCheck(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	writeClaudeSkill(t, home, "databricks-core")
	asked := setInstallsFor(t, true)
	ctx, _ := claudeCtx(t, home)
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, *asked)
}
