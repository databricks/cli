package agents

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentHintText(t *testing.T) {
	const prefix = "Databricks skills are not installed. To work with Databricks reliably, first run: "
	cases := map[string]string{
		NameCursor:      "databricks aitools install --agents cursor --scope global",
		NameCodex:       "databricks aitools install --agents codex --scope global",
		NameOpenCode:    "databricks aitools install --agents opencode --scope global",
		NameCopilot:     "databricks aitools install --agents copilot --scope global",
		NameAntigravity: "databricks aitools install --agents antigravity --scope global",
		NamePi:          "databricks aitools install --agents pi --scope global",
		NameGemini:      "databricks aitools install --agents gemini --scope global",
		NameGoose:       "databricks aitools install --agents goose --scope global",
		NameKiro:        "databricks aitools install --agents kiro --scope global",
	}
	for name, command := range cases {
		t.Run(name, func(t *testing.T) {
			a := ByName(name)
			require.NotNil(t, a)
			assert.Equal(t, prefix+command, agentInstallHint(a, noPluginRecorded).text)
		})
	}
}

func TestAgentInstallHintThrottleKey(t *testing.T) {
	codex := agentInstallHint(ByName(NameCodex), noPluginRecorded)
	cursor := agentInstallHint(ByName(NameCursor), noPluginRecorded)

	ctx := env.Set(t.Context(), "CODEX_THREAD_ID", "thread-1")
	assert.Equal(t, "codex:session:thread-1", codex.throttleKey(ctx))

	// No session id: fall back to one key per agent on this machine.
	assert.Equal(t, "codex:machine", codex.throttleKey(env.Set(t.Context(), "CODEX_THREAD_ID", "")))

	// Agents without a verified session variable always use the per-agent key,
	// and never pick up another agent's session id.
	assert.Equal(t, "cursor:machine", cursor.throttleKey(ctx))
}

// agentDetectEnv is the SDK env var that makes AgentProvider() report each
// registry agent.
var agentDetectEnv = map[string]string{
	NameClaudeCode:  "CLAUDECODE",
	NameCursor:      "CURSOR_AGENT",
	NameCodex:       "CODEX_CI",
	NameOpenCode:    "OPENCODE",
	NameCopilot:     "COPILOT_CLI",
	NameAntigravity: "ANTIGRAVITY_AGENT",
	NamePi:          "PI_CODING_AGENT",
	NameGemini:      "GEMINI_CLI",
	NameGoose:       "GOOSE_TERMINAL",
	NameKiro:        "KIRO",
}

// setAgentEnv makes useragent.AgentProvider() report the given env var's agent,
// with the host's agent and hint env cleared (see testutil.SetAgentEnv).
func setAgentEnv(t *testing.T, envVar string) {
	testutil.SetAgentEnv(t, envVar, "1")
	unsetEnv(t, append([]string{"CODEX_THREAD_ID"}, hintEnvVars...))
}

// agentCtx returns a hint test context whose agent config dirs all resolve
// under home, so fixtures never touch the host's real agent directories. It
// also changes into a repository root under home, so project skills around
// the directory the tests run from are never read.
func agentCtx(t *testing.T, home, cacheDir string) (context.Context, *bytes.Buffer) {
	project := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".git"), 0o755))
	t.Chdir(project)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.WithUserHomeDir(ctx, home)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)
	// The throttle marker lives in the file cache, which a host could disable.
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "CLAUDE_CONFIG_DIR", "")
	ctx = env.Set(ctx, "XDG_CONFIG_HOME", "")
	ctx = env.Set(ctx, "APPDATA", "")
	ctx = env.Set(ctx, "GEMINI_CLI_HOME", "")
	ctx = env.Set(ctx, "PI_CODING_AGENT_DIR", "")
	ctx = env.Set(ctx, "GOOSE_PATH_ROOT", "")
	ctx = env.Set(ctx, "CODEX_THREAD_ID", "")
	return ctx, stderr
}

func TestAgentCtxIgnoresProjectSkillsAroundTheTest(t *testing.T) {
	// Project skills in the directory the tests run from (e.g. the checkout)
	// must not count as installed.
	gemini := ByName(NameGemini)
	outer := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(outer, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(gemini.ProjectSkillsDir(outer), "databricks-core"), 0o755))
	t.Chdir(outer)

	setAgentEnv(t, agentDetectEnv[NameGemini])
	ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Equal(t, agentInstallHint(gemini, noPluginRecorded).text+"\n", stderr.String())
}

func writeAgentSkill(t *testing.T, ctx context.Context, home string, a *Agent) {
	dir, err := a.SkillsDir(ctx)
	require.NoError(t, err)
	rel, err := filepath.Rel(home, dir)
	require.NoError(t, err)
	require.False(t, strings.HasPrefix(rel, ".."), "skills dir %s escapes the test home", dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "databricks-core"), 0o755))
}

func TestMaybeHint_PerAgent(t *testing.T) {
	for _, a := range Registry {
		if a.Name == NameClaudeCode {
			continue // Claude's hint is covered by hint_test.go.
		}
		t.Run(a.Name, func(t *testing.T) {
			setAgentEnv(t, agentDetectEnv[a.Name])

			t.Run("not installed hints", func(t *testing.T) {
				ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
				ctx = telemetry.WithNewLogger(ctx)
				MaybeHint(ctx, regularCmd(), noPluginRecorded)
				assert.Equal(t, agentInstallHint(a, noPluginRecorded).text+"\n", stderr.String())

				events := telemetry.BufferedEvents(ctx)
				require.Len(t, events, 1)
				assert.Equal(t, &protos.AitoolsHintEvent{
					HintKind: protos.AitoolsHintKindAgentInstallText,
				}, events[0].AitoolsHintEvent)
			})

			t.Run("own skills installed is silent", func(t *testing.T) {
				home := t.TempDir()
				ctx, stderr := agentCtx(t, home, t.TempDir())
				ctx = telemetry.WithNewLogger(ctx)
				writeAgentSkill(t, ctx, home, a)
				MaybeHint(ctx, regularCmd(), noPluginRecorded)
				assert.Empty(t, stderr.String())
				assert.Empty(t, telemetry.BufferedEvents(ctx))
			})

			t.Run("canonical skills alone still hint", func(t *testing.T) {
				// Skills in the shared store are not loaded by the agent unless
				// they are also linked into its own skills dir.
				home := t.TempDir()
				ctx, stderr := agentCtx(t, home, t.TempDir())
				writeCanonicalSkill(t, home, "databricks")
				MaybeHint(ctx, regularCmd(), noPluginRecorded)
				assert.Equal(t, agentInstallHint(a, noPluginRecorded).text+"\n", stderr.String())
			})

			t.Run("throttled per machine without session id", func(t *testing.T) {
				home, cacheDir := t.TempDir(), t.TempDir()
				ctx1, stderr1 := agentCtx(t, home, cacheDir)
				MaybeHint(ctx1, regularCmd(), noPluginRecorded)
				assert.NotEmpty(t, stderr1.String())

				ctx2, stderr2 := agentCtx(t, home, cacheDir)
				MaybeHint(ctx2, regularCmd(), noPluginRecorded)
				assert.Empty(t, stderr2.String())
			})
		})
	}
}

// noPluginRecorded is the CLI install record of a machine where `databricks
// aitools install` recorded no plugin.
func noPluginRecorded(context.Context, string) bool { return false }

func TestMaybeHint_RecordedPluginSilent(t *testing.T) {
	// Codex and Copilot have no plugin manifest reader: the CLI install record
	// is their plugin source.
	for _, name := range []string{NameCodex, NameCopilot} {
		t.Run(name, func(t *testing.T) {
			setAgentEnv(t, agentDetectEnv[name])
			var asked []string
			recorded := func(_ context.Context, agentName string) bool {
				asked = append(asked, agentName)
				return agentName == name
			}
			ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
			ctx = telemetry.WithNewLogger(ctx)
			MaybeHint(ctx, regularCmd(), recorded)
			assert.Empty(t, stderr.String())
			assert.Empty(t, telemetry.BufferedEvents(ctx))
			assert.Equal(t, []string{name}, asked)
		})
	}
}

func TestMaybeHint_CodexThrottledPerThread(t *testing.T) {
	setAgentEnv(t, "CODEX_CI")
	home, cacheDir := t.TempDir(), t.TempDir()
	run := func(thread string) string {
		ctx, stderr := agentCtx(t, home, cacheDir)
		ctx = env.Set(ctx, "CODEX_THREAD_ID", thread)
		MaybeHint(ctx, regularCmd(), noPluginRecorded)
		return stderr.String()
	}

	assert.Contains(t, run("thread-1"), "--agents codex")
	assert.Empty(t, run("thread-1"))
	// A new Codex session is hinted again even within the TTL.
	assert.Contains(t, run("thread-2"), "--agents codex")
}

func TestMaybeHint_NoInstallTargetSilent(t *testing.T) {
	for _, envVar := range []string{"AMP_CURRENT_THREAD_ID", "AUGMENT_AGENT", "CLINE_ACTIVE", "OPENCLAW_SHELL", "VSCODE_AGENT", "WINDSURF_AGENT"} {
		t.Run(envVar, func(t *testing.T) {
			setAgentEnv(t, envVar)
			ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
			MaybeHint(ctx, regularCmd(), noPluginRecorded)
			assert.Empty(t, stderr.String())
		})
	}
}

func TestMaybeHint_MultipleAgentsSilent(t *testing.T) {
	setAgentEnv(t, "CODEX_CI")
	t.Setenv("CURSOR_AGENT", "1")
	ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_AgentAuthTokenSilent(t *testing.T) {
	setAgentEnv(t, "CODEX_CI")
	ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
	parent := &cobra.Command{Use: authCommandName}
	child := &cobra.Command{Use: tokenCommandName}
	parent.AddCommand(child)
	MaybeHint(ctx, child, noPluginRecorded)
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_AgentAIToolsCommandSilent(t *testing.T) {
	setAgentEnv(t, "CODEX_CI")
	ctx, stderr := agentCtx(t, t.TempDir(), t.TempDir())
	parent := &cobra.Command{Use: aiToolsCommandName}
	child := &cobra.Command{Use: "install"}
	parent.AddCommand(child)
	MaybeHint(ctx, child, noPluginRecorded)
	assert.Empty(t, stderr.String())
}

func TestMaybeHint_SessionMarkersAreIsolatedPerAgent(t *testing.T) {
	// Claude and Codex both report the same session id. A Claude marker must not
	// silence Codex's first hint.
	home, cacheDir := t.TempDir(), t.TempDir()

	setClaudeAgent(t)
	ctx, stderr := agentCtx(t, home, cacheDir)
	ctx = env.Set(ctx, claudeSessionEnv, "shared-id")
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	require.Contains(t, stderr.String(), hintText)

	setAgentEnv(t, "CODEX_CI")
	ctx, stderr = agentCtx(t, home, cacheDir)
	ctx = env.Set(ctx, "CODEX_THREAD_ID", "shared-id")
	MaybeHint(ctx, regularCmd(), noPluginRecorded)
	assert.Equal(t, agentInstallHint(ByName(NameCodex), noPluginRecorded).text+"\n", stderr.String())
}

func TestMaybeHint_UnwritableCacheSkipsHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not make a directory read-only on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}
	setAgentEnv(t, "CODEX_CI")
	readOnly := t.TempDir()
	require.NoError(t, os.Chmod(readOnly, 0o555))
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	// Without a kept marker the hint would repeat on every command, so none is
	// printed, on every call, and no other store (e.g. under TMPDIR) is used.
	for range 2 {
		ctx, stderr := agentCtx(t, t.TempDir(), readOnly)
		ctx = telemetry.WithNewLogger(ctx)
		ctx = env.Set(ctx, "CODEX_THREAD_ID", "thread-1")
		MaybeHint(ctx, regularCmd(), noPluginRecorded)
		assert.Empty(t, stderr.String())
		assert.Empty(t, telemetry.BufferedEvents(ctx))
	}
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
