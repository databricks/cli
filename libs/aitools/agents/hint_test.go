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

// claudeCtx returns a context with a captured, non-TTY stderr buffer (what a
// coding agent's shell tool gives the CLI) and an isolated HOME + cache dir,
// so detection and the session throttle read only the test's temp state. home
// is returned so the test can drop fixtures under it.
func claudeCtx(t *testing.T, home string) (context.Context, *bytes.Buffer) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
	// Throttle markers live in the file cache, which a host could disable.
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
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

// writeProjectPluginManifest writes a manifest with one project-scope install
// covering projectPath. Marshaled, not interpolated, so Windows paths with
// backslashes stay valid JSON.
func writeProjectPluginManifest(t *testing.T, home, projectPath string) {
	manifest, err := json.Marshal(map[string]any{
		"plugins": map[string]any{
			"databricks@m": []any{map[string]string{
				"scope":       "project",
				"projectPath": projectPath,
			}},
		},
	})
	require.NoError(t, err)
	writePluginManifest(t, home, string(manifest))
}

// writeClaudeSkill puts a skill dir in Claude Code's own skills dir. It is a
// real dir rather than the installer's symlink so the fixture needs no symlink
// privilege on Windows; skill detection treats both alike.
func writeClaudeSkill(t *testing.T, home, name string) {
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills", name), 0o755))
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
// kind INSTALL_TEXT.
func assertHinted(t *testing.T, ctx context.Context, stderr *bytes.Buffer) {
	t.Helper()
	assert.Equal(t, airHintText+"\n", stderr.String())
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx))
}

// assertSilent checks for no hint line and no telemetry event.
func assertSilent(t *testing.T, ctx context.Context, stderr *bytes.Buffer) {
	t.Helper()
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestAIRHintText(t *testing.T) {
	// The hint names the skill, says what it is for, and offers both the install
	// and a no-install preview, on one line.
	assert.Contains(t, airHintText, "install the databricks-ai-runtime skill")
	assert.Contains(t, airHintText, "workload configuration and run debugging")
	assert.Contains(t, airHintText, "databricks aitools install --skills-only --skills databricks-ai-runtime")
	assert.Contains(t, airHintText, "replace --skills-only with --path <dir>")
	assert.NotContains(t, airHintText, "\n")
}

func TestMaybeHint_HumanCallerSilent(t *testing.T) {
	// No agent env: AgentProvider() == "", so no hint even with nothing installed.
	testutil.SetAgentEnv(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assertSilent(t, ctx, stderr)
}

// Every air subcommand, at any depth, gets the skill-specific hint.
func TestMaybeHint_AIRSubcommandsHint(t *testing.T) {
	setClaudeAgent(t)
	for _, path := range []string{
		"air run", "air get", "air list", "air logs", "air cancel",
		"air convert-to-dabs", "air images", "air images push", "air pools", "air pools list",
	} {
		t.Run(path, func(t *testing.T) {
			ctx, stderr := claudeCtx(t, t.TempDir())
			MaybeHint(ctx, commandAt(path), noPluginRecorded)
			assertHinted(t, ctx, stderr)
		})
	}
}

// Commands outside air are never hinted, including the auth and version
// commands the generic hint used to fire on.
func TestMaybeHint_NonAIRCommandsSilent(t *testing.T) {
	setClaudeAgent(t)
	for _, path := range []string{
		"auth profiles", "auth token", "current-user me", "clusters list", "jobs list",
		"bundle deploy", "workspace list", "version", "aitools install",
		"experimental aitools install",
		// The bare air node only renders help; its footer carries the tip.
		"air",
		// The hidden alias that points users at `databricks air`.
		"experimental air",
	} {
		t.Run(path, func(t *testing.T) {
			ctx, stderr := claudeCtx(t, t.TempDir())
			MaybeHint(ctx, commandAt(path), noPluginRecorded)
			assertSilent(t, ctx, stderr)
		})
	}
}

func TestMaybeHint_ClaudeNotInstalledHints(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())
	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assert.Contains(t, stderr.String(), airHintText)
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindInstallText}, hintKinds(t, ctx))
}

func TestMaybeHint_PluginInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	// A plugin install writes only Claude's manifest, no skills subdir, and
	// bundles every Databricks skill including databricks-ai-runtime.
	writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)

	MaybeHint(ctx, airCmd(), noPluginRecorded)
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

// A project or local plugin only loads inside its own project, so elsewhere
// Claude Code still lacks the skill and gets the hint.
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

			MaybeHint(ctx, airCmd(), noPluginRecorded)
			if tc.wantHint {
				assertHinted(t, ctx, stderr)
			} else {
				assertSilent(t, ctx, stderr)
			}
		})
	}
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

			MaybeHint(ctx, airCmd(), noPluginRecorded)
			if tc.wantHint {
				assertHinted(t, ctx, stderr)
			} else {
				assertSilent(t, ctx, stderr)
			}
		})
	}
}

func TestMaybeHint_SkillsInstalledSilent(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeClaudeSkill(t, home, airSkillName)

	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assertSilent(t, ctx, stderr)
}

func TestMaybeHint_SharedStoreOnlyStillHints(t *testing.T) {
	// Skills in the shared store (or the legacy store) but not in Claude Code's
	// own skills dir were installed for another agent; Claude doesn't load them.
	for _, dir := range []string{CanonicalSkillsDir, LegacySkillsDir} {
		t.Run(dir, func(t *testing.T) {
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			require.NoError(t, os.MkdirAll(filepath.Join(home, dir, airSkillName), 0o755))

			MaybeHint(ctx, airCmd(), noPluginRecorded)
			assert.Equal(t, airHintText+"\n", stderr.String())
		})
	}
}

func TestMaybeHint_NonDatabricksSkillStillHints(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeClaudeSkill(t, home, "some-other-skill")

	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assert.Contains(t, stderr.String(), airHintText)
}

// Other Databricks skills, even in Claude Code's own skills dir, do not
// silence the hint: they don't cover air, and the hint recommends installing
// exactly the AI Runtime skill.
func TestMaybeHint_OtherDatabricksSkillStillHints(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	writeClaudeSkill(t, home, "databricks")
	writeClaudeSkill(t, home, "databricks-jobs")

	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assertHinted(t, ctx, stderr)
}

func TestMaybeHint_AIToolsCommandSilent(t *testing.T) {
	setClaudeAgent(t)
	ctx, stderr := claudeCtx(t, t.TempDir())

	parent := &cobra.Command{Use: aiToolsCommandName}
	child := &cobra.Command{Use: "install"}
	parent.AddCommand(child)

	MaybeHint(ctx, child, noPluginRecorded)
	assertSilent(t, ctx, stderr)
}

func TestMaybeHint_NoIONoPanic(t *testing.T) {
	setClaudeAgent(t)
	// Bare context without cmdio installed: cobra resolves --help and bare
	// invocations before PersistentPreRunE runs, so HasIO can be false here.
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	ctx = telemetry.WithNewLogger(ctx)
	assert.NotPanics(t, func() { MaybeHint(ctx, airCmd(), noPluginRecorded) })
	assert.Empty(t, hintKinds(t, ctx))
}

// With a session id the first air command in the session is hinted and later
// ones stay quiet until the marker expires; without one (older Claude Code)
// every air command is hinted. Both cases run with a non-TTY stderr, which is
// how a coding agent's shell tool runs the CLI.
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
			MaybeHint(ctx0, commandAt("auth profiles"), noPluginRecorded)
			assertSilent(t, ctx0, stderr0)

			ctx1, stderr1 := newCtx()
			MaybeHint(ctx1, commandAt("air list"), noPluginRecorded)
			assertHinted(t, ctx1, stderr1)

			ctx2, stderr2 := newCtx()
			MaybeHint(ctx2, commandAt("air run"), noPluginRecorded)
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
		MaybeHint(ctx, airCmd(), noPluginRecorded)
		assertHinted(t, ctx, stderr)
	}
}

// Only the databricks-ai-runtime skill itself, in a location Claude Code loads
// for this working directory, silences the hint.
func TestMaybeHint_AIRSkillDetection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		setup    func(t *testing.T, home, cwd string)
		wantHint bool
	}{
		{"nothing installed", func(t *testing.T, home, cwd string) {}, true},
		{"skill in Claude's own skills dir", func(t *testing.T, home, cwd string) {
			writeClaudeSkill(t, home, airSkillName)
		}, false},
		{"skill in the project skills dir", func(t *testing.T, home, cwd string) {
			require.NoError(t, os.MkdirAll(filepath.Join(cwd, ".claude", "skills", airSkillName), 0o755))
		}, false},
		{"other databricks skills only", func(t *testing.T, home, cwd string) {
			writeClaudeSkill(t, home, "databricks")
			writeClaudeSkill(t, home, "databricks-jobs")
		}, true},
		{"a file named like the skill", func(t *testing.T, home, cwd string) {
			dir := filepath.Join(home, ".claude", "skills")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, airSkillName), nil, 0o600))
		}, true},
		{"shared canonical store", func(t *testing.T, home, cwd string) {
			writeCanonicalSkill(t, home, airSkillName)
		}, true},
		{"shared legacy store", func(t *testing.T, home, cwd string) {
			require.NoError(t, os.MkdirAll(filepath.Join(home, LegacySkillsDir, airSkillName), 0o755))
		}, true},
		{"plugin for this project", func(t *testing.T, home, cwd string) {
			writeProjectPluginManifest(t, home, cwd)
		}, false},
		{"plugin for an unrelated project", func(t *testing.T, home, cwd string) {
			writeProjectPluginManifest(t, home, t.TempDir())
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A fresh project per case: the project-skills fixtures write into
			// the working directory and must not leak between cases.
			setClaudeAgent(t)
			home := t.TempDir()
			ctx, stderr := claudeCtx(t, home)
			cwd, err := os.Getwd()
			require.NoError(t, err)
			tc.setup(t, home, cwd)

			MaybeHint(ctx, airCmd(), noPluginRecorded)
			if tc.wantHint {
				assertHinted(t, ctx, stderr)
			} else {
				assertSilent(t, ctx, stderr)
			}
		})
	}
}

// A project-scoped skill at the repository root still counts when the CLI runs
// in a subdirectory, as Claude Code loads it there.
func TestMaybeHint_AIRSkillInRepositoryRoot(t *testing.T) {
	setClaudeAgent(t)
	home := t.TempDir()
	ctx, stderr := claudeCtx(t, home)
	root, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".claude", "skills", airSkillName), 0o755))
	sub := filepath.Join(root, "training", "configs")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	t.Chdir(sub)

	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assertSilent(t, ctx, stderr)
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
	auth := &cobra.Command{Use: "auth"}
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
			wantStderr: wantNativeHint + "\n" + airHintText + "\n",
		},
		{
			name:       "no child session prints text only",
			wantStderr: airHintText + "\n",
		},
		{
			name:       "CI prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "true"},
			wantStderr: airHintText + "\n",
		},
		{
			name:       "CI=false still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "false"},
			wantStderr: wantNativeHint + "\n" + airHintText + "\n",
		},
		{
			name:       "CI=no still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "no"},
			wantStderr: wantNativeHint + "\n" + airHintText + "\n",
		},
		{
			name:       "CI=OFF still prints native",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "OFF"},
			wantStderr: wantNativeHint + "\n" + airHintText + "\n",
		},
		{
			// Some providers set CI to their own name.
			name:       "CI set to a provider name prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", ciEnv: "woodpecker"},
			wantStderr: airHintText + "\n",
		},
		{
			// Azure Pipelines sets TF_BUILD but not CI.
			name:       "CI provider without CI var prints text only",
			env:        map[string]string{claudeChildSessionEnv: "1", "TF_BUILD": "True"},
			wantStderr: airHintText + "\n",
		},
		{
			name: "plugin installed prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			setup: func(t *testing.T, home string) {
				writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user"}]}}`)
			},
		},
		{
			name: "skill installed prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			setup: func(t *testing.T, home string) {
				writeClaudeSkill(t, home, airSkillName)
			},
		},
		{
			// Commands outside air never hint, credential helpers included.
			name: "auth token prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd: func() *cobra.Command {
				parent := &cobra.Command{Use: "auth"}
				child := &cobra.Command{Use: "token [PROFILE]"}
				parent.AddCommand(child)
				return child
			},
		},
		{
			name: "other auth command prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd: func() *cobra.Command {
				parent := &cobra.Command{Use: "auth"}
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
			name: "other auth docker command prints nothing",
			env:  map[string]string{claudeChildSessionEnv: "1"},
			cmd:  func() *cobra.Command { return authDockerCmd("configure [PROFILE]") },
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
			cmd := airCmd()
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

			MaybeHint(ctx, airCmd(), noPluginRecorded)
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
		return telemetry.WithNewLogger(ctx), stderr
	}

	ctx1, stderr1 := newCtx()
	MaybeHint(ctx1, airCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+airHintText+"\n", stderr1.String())

	// Same session, second command: both lines stay quiet, so the tag doesn't
	// reach a downstream reader such as `2>&1 | jq` on every command.
	ctx2, stderr2 := newCtx()
	MaybeHint(ctx2, airCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())

	// Another session: both lines print again.
	ctx3, stderr3 := newCtx()
	ctx3 = env.Set(ctx3, claudeSessionEnv, "session-def")
	MaybeHint(ctx3, airCmd(), noPluginRecorded)
	assert.Equal(t, wantNativeHint+"\n"+airHintText+"\n", stderr3.String())
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

			MaybeHint(ctx, airCmd(), noPluginRecorded)
			assert.Equal(t, airHintText+"\n", stderr.String())
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
	assert.Equal(t, airHintText, claude.text)
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
	MaybeHint(ctx1, airCmd(), noPluginRecorded)
	assert.Equal(t, []protos.AitoolsHintKind{
		protos.AitoolsHintKindClaudeCodeNative,
		protos.AitoolsHintKindInstallText,
	}, hintKinds(t, ctx1))
	events := telemetry.BufferedEvents(ctx1)
	require.Len(t, events, 2)

	// Throttled for the session: nothing prints and nothing logs, so the
	// command doesn't pay for a telemetry upload at exit.
	ctx2, stderr2 := newCtx(true)
	MaybeHint(ctx2, airCmd(), noPluginRecorded)
	assert.Empty(t, stderr2.String())
	assert.Empty(t, telemetry.BufferedEvents(ctx2))

	// Throttled and not a Claude Code subprocess: nothing prints, nothing logs.
	ctx3, stderr3 := newCtx(false)
	MaybeHint(ctx3, airCmd(), noPluginRecorded)
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

	MaybeHint(ctx, airCmd(), noPluginRecorded)
	assert.Empty(t, telemetry.BufferedEvents(ctx))
}
