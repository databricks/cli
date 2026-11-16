package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dnsError is what a fetch that got no response wraps.
var dnsError = &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: "raw.githubusercontent.com", IsNotFound: true}}

func TestInstallRecovery(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "")
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "release lookup rate limited",
			err:  fmt.Errorf("resolve latest skills release: %w", &installer.FetchError{Stage: installer.FetchStageRelease, StatusCode: http.StatusForbidden, RateLimited: true, Err: errors.New("HTTP 403 fetching latest skills release")}),
			want: "api.github.com rate-limited the lookup of the latest Databricks skills release. Wait a few minutes and re-run 'databricks aitools install', or skip the lookup by setting DATABRICKS_SKILLS_REF to a release tag from https://github.com/databricks/databricks-agent-skills/releases.",
		},
		{
			name: "release lookup denied without rate-limit evidence",
			err:  &installer.FetchError{Stage: installer.FetchStageRelease, StatusCode: http.StatusForbidden, Err: errors.New("HTTP 403 fetching latest skills release")},
			want: "Could not look up the latest Databricks skills release on api.github.com. Wait a few minutes and re-run 'databricks aitools install', or skip the lookup by setting DATABRICKS_SKILLS_REF to a release tag from https://github.com/databricks/databricks-agent-skills/releases.",
		},
		{
			name: "release lookup network error",
			err:  &installer.FetchError{Stage: installer.FetchStageRelease, Err: dnsError},
			want: "Could not look up the latest Databricks skills release on api.github.com. Wait a few minutes and re-run 'databricks aitools install', or skip the lookup by setting DATABRICKS_SKILLS_REF to a release tag from https://github.com/databricks/databricks-agent-skills/releases.",
		},
		{
			name: "manifest fetch server error",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusServiceUnavailable, Err: errors.New("failed to fetch manifest: HTTP 503")},
			want: "raw.githubusercontent.com returned HTTP 503, a server-side error. Wait a few minutes and re-run 'databricks aitools install'.",
		},
		{
			name: "manifest fetch refused without rate-limit evidence",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusForbidden, Err: errors.New("failed to fetch manifest: HTTP 403")},
			want: "raw.githubusercontent.com refused the download (HTTP 403). A proxy or access policy may block it; ask your network admin to allow raw.githubusercontent.com, then re-run 'databricks aitools install'.",
		},
		{
			name: "manifest fetch rate limited",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusTooManyRequests, RateLimited: true, Err: errors.New("failed to fetch manifest: HTTP 429")},
			want: "raw.githubusercontent.com rate-limited the download (HTTP 429). Wait a few minutes and re-run 'databricks aitools install'.",
		},
		{
			name: "manifest is not JSON",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusOK, Err: fmt.Errorf("failed to parse manifest: %w", &json.SyntaxError{})},
			want: "raw.githubusercontent.com answered with something that is not a skills manifest (a proxy or captive portal page, or a broken mirror). Check what this machine gets from raw.githubusercontent.com, then re-run 'databricks aitools install'.",
		},
		{
			name: "manifest fetch without a response",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, Err: fmt.Errorf("failed to fetch manifest: %w", dnsError)},
			want: "Could not download Databricks skills from raw.githubusercontent.com. Check that this machine can reach it (set HTTPS_PROXY if you use a proxy), then re-run 'databricks aitools install'.",
		},
		{
			name: "body read failure after HTTP 200 is a network failure",
			err:  &installer.FetchError{Stage: installer.FetchStageFile, StatusCode: http.StatusOK, Err: fmt.Errorf("failed to read response: %w", io.ErrUnexpectedEOF)},
			want: "Could not download Databricks skills from raw.githubusercontent.com. Check that this machine can reach it (set HTTPS_PROXY if you use a proxy), then re-run 'databricks aitools install'.",
		},
		{
			name: "manifest missing at the cli-compat pin",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusNotFound, Err: fmt.Errorf("skills manifest at databricks-agent-skills@v9: %w", clicompat.ErrNotFound)},
			want: "The skills release this CLI selected is missing. Set DATABRICKS_SKILLS_REF to a release tag from https://github.com/databricks/databricks-agent-skills/releases and re-run 'databricks aitools install', and report it at https://github.com/databricks/cli/issues.",
		},
		{
			name: "skill file fetch",
			err:  fmt.Errorf("failed to fetch skills/x/SKILL.md at v1: %w", &installer.FetchError{Stage: installer.FetchStageFile, StatusCode: http.StatusNotFound, Err: errors.New("HTTP 404")}),
			want: "raw.githubusercontent.com returned HTTP 404. Re-run 'databricks aitools install --debug' to see details, and report it at https://github.com/databricks/cli/issues.",
		},
		{
			name: "corrupt state file",
			err:  fmt.Errorf("failed to load install state: %w", &installer.StateError{Path: filepath.FromSlash("/home/u/.databricks/aitools/skills/.state.json"), Err: errors.New("failed to parse state file")}),
			want: "Move the install state file /home/u/.databricks/aitools/skills/.state.json aside (for example, rename it to .state.json.bak), then re-run 'databricks aitools install'.",
		},
		{
			name: "unreadable state file keeps the permission step",
			err:  fmt.Errorf("failed to load install state: %w", fmt.Errorf("failed to read state file: %w", &fs.PathError{Op: "open", Path: filepath.FromSlash("/home/u/.databricks/aitools/skills/.state.json"), Err: fs.ErrPermission})),
			want: "Check that you own /home/u/.databricks/aitools/skills/.state.json and can write to it (an earlier run with sudo can leave it owned by root), then re-run 'databricks aitools install'.",
		},
		{
			name: "permission denied",
			err:  fmt.Errorf("failed to create skills directory: %w", &fs.PathError{Op: "mkdir", Path: filepath.FromSlash("/home/u/.databricks/aitools"), Err: fs.ErrPermission}),
			want: "Check that you own /home/u/.databricks/aitools and can write to it (an earlier run with sudo can leave it owned by root), then re-run 'databricks aitools install'.",
		},
		{
			name: "permission denied on a rename names its destination",
			err:  fmt.Errorf("failed to move skill into place: %w", &os.LinkError{Op: "rename", Old: filepath.FromSlash("/home/u/.tmp/x"), New: filepath.FromSlash("/home/u/.databricks/aitools/skills/x"), Err: fs.ErrPermission}),
			want: "Check that you own /home/u/.databricks/aitools/skills/x and can write to it (an earlier run with sudo can leave it owned by root), then re-run 'databricks aitools install'.",
		},
		{
			name: "permission denied without a path",
			err:  os.NewSyscallError("open", syscall.EACCES),
			want: "Check that you own the directories the install writes to and can write to them (an earlier run with sudo can leave them owned by root), then re-run 'databricks aitools install'.",
		},
		{
			name: "prompt aborted",
			err:  promptError(huh.ErrUserAborted),
			want: "To install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=<name>",
		},
		{
			name: "nothing selected in picker",
			err:  errNoAgentsSelected,
			want: "To install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=<name>",
		},
		{
			name: "terminal that cannot be opened is a prompt failure, not a permission problem",
			err:  promptError(&fs.PathError{Op: "open", Path: "/dev/tty", Err: fs.ErrPermission}),
			want: "To install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=<name>",
		},
		{
			name: "skill not found",
			err:  &installer.SkillError{Skill: "x", Reason: installer.ReasonSkillNotFound, Detail: "not found"},
			want: "Run 'databricks aitools list' to see the available skills.",
		},
		{
			name: "experimental skill already names the flag",
			err:  &installer.SkillError{Skill: "x", Reason: installer.ReasonExperimentalSkill, Detail: "is experimental; use --experimental to install"},
			want: "",
		},
		{
			name: "interrupted command gets no step",
			err:  &installer.FetchError{Stage: installer.FetchStageManifest, Err: fmt.Errorf("failed to fetch manifest: %w", context.Canceled)},
			want: "",
		},
		{
			name: "refused downgrade already says what to do",
			err:  &installer.RefusedError{Err: errors.New("refusing to downgrade skills from 0.3.0 to 0.2.0; ... set DATABRICKS_SKILLS_REF=v0.2.0 to select this version explicitly")},
			want: "",
		},
		{
			name: "legacy install guard already says what to do",
			err:  &installer.RefusedError{Err: errors.New("legacy install detected without state tracking; run 'databricks aitools install' (without a skill name) first to rebuild state")},
			want: "",
		},
		{
			name: "per-agent failures carry their own steps",
			err:  &agentErrors{err: errors.New("Cursor: does not support project-scoped skills")},
			want: "",
		},
		{
			name: "no agent set up, each agent's step is printed above",
			err:  &agentErrors{err: errNoAgentSetUp},
			want: "",
		},
		{
			name: "dump wrote no skill",
			err:  installer.ErrNoSkillsResolved,
			want: "No skill in this release can be installed by this CLI. Re-run 'databricks aitools install --debug' to see why each skill was skipped. To include experimental skills, add --experimental.",
		},
		{
			name: "unknown error",
			err:  errors.New("boom"),
			want: "Re-run 'databricks aitools install --debug' to see details. If it keeps failing, report it at https://github.com/databricks/cli/issues.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, installRecovery(t.Context(), installInvocation{}, tc.err))
		})
	}
}

func TestNoSkillsWrittenRecoveryWithExperimental(t *testing.T) {
	// --experimental was already given, so the step does not suggest it again.
	inv := installInvocation{path: "dump", experimental: true}
	assert.Equal(t, "No skill in this release can be installed by this CLI. Re-run 'databricks aitools install --path=dump --experimental --debug' to see why each skill was skipped.", installRecovery(t.Context(), inv, installer.ErrNoSkillsResolved))
}

func TestWithRecoveryKeepsCauseAndAppendsStep(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "")
	cause := &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusBadGateway, Err: errors.New("failed to fetch manifest: HTTP 502")}
	err := withRecovery(t.Context(), installInvocation{}, cause)

	assert.Equal(t, "failed to fetch manifest: HTTP 502\nraw.githubusercontent.com returned HTTP 502, a server-side error. Wait a few minutes and re-run 'databricks aitools install'", err.Error())
	_, ok := errors.AsType[*installer.FetchError](err)
	assert.True(t, ok, "classification must still see the cause")
	assert.NoError(t, withRecovery(t.Context(), installInvocation{}, nil))
}

func TestFetchRecoveryNamesOverriddenHost(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "https://mirror.example.test/skills")
	got := installRecovery(t.Context(), installInvocation{}, &installer.FetchError{Stage: installer.FetchStageManifest, Err: dnsError})
	assert.Contains(t, got, "Could not download Databricks skills from mirror.example.test.")
}

func TestReleaseLookupRecoveryNamesOverriddenHost(t *testing.T) {
	t.Setenv("DATABRICKS_CLI_GITHUB_API_URL", "https://ghe.example.test/api/v3")
	got := installRecovery(t.Context(), installInvocation{}, &installer.FetchError{Stage: installer.FetchStageRelease, Err: dnsError})
	assert.Contains(t, got, "Could not look up the latest Databricks skills release on ghe.example.test.")
}

func TestPluginFailureText(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	claude.Plugin.Source = "anthropics/claude-plugins-official"
	claude.Plugin.Marketplace = "claude-plugins-official"
	item := agentPlanItem{agent: claude, delivery: deliveryPlugin, scope: agentScopeUser}
	inv := installInvocation{scope: installer.ScopeGlobal, agents: agents.NameClaudeCode}
	const rerun = "databricks aitools install --scope=global --agents=claude-code"

	tests := []struct {
		name         string
		err          error
		wantReason   string
		wantRecovery string
	}{
		{
			name:         "cli not on path",
			err:          &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: `could not resolve claude on PATH: exec: "claude": executable file not found in $PATH`},
			wantReason:   "'claude' is not on PATH",
			wantRecovery: "Install or update the Claude Code CLI so 'claude' is on PATH, then re-run '" + rerun + "'. Or install skill files instead: " + rerun + " --skills-only",
		},
		{
			name:         "install failed",
			err:          &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonInstallFailed, Detail: "fatal: unable to access github.com"},
			wantReason:   "'claude plugin' failed: fatal: unable to access github.com",
			wantRecovery: "Run 'claude plugin marketplace add anthropics/claude-plugins-official', then 'claude plugin marketplace update claude-plugins-official', then 'claude plugin install databricks@claude-plugins-official --scope user' to see the full error. Once they succeed, re-run '" + rerun + "' so the CLI records the plugin. Or install skill files instead: " + rerun + " --skills-only",
		},
		{
			name:         "timed out",
			err:          &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: "timed out after 20s", TimedOut: true},
			wantReason:   "'claude plugin' timed out after 20s",
			wantRecovery: "Re-run '" + rerun + "' (an agent CLI's first start can be slow), or install skill files instead: " + rerun + " --skills-only",
		},
		{
			name:         "windows shim exited with an error",
			err:          &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonShimExitedWithError, Detail: "printed its plugin help but exited with an error (exit status 1): node: bad option; to see the error and exit code, run ..."},
			wantReason:   "'claude plugin --help' printed its plugin help but exited with an error (exit status 1): node: bad option; to see the error and exit code, run ...",
			wantRecovery: "Once it exits with 0, re-run '" + rerun + "'. Or install skill files instead: " + rerun + " --skills-only",
		},
		{
			name:         "untyped error",
			err:          errors.New("boom"),
			wantReason:   "boom",
			wantRecovery: "Re-run '" + rerun + " --debug' to see details, or install skill files instead: " + rerun + " --skills-only",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason, recovery := pluginFailureText(t.Context(), inv, item, tc.err)
			assert.Equal(t, tc.wantReason, reason)
			assert.Equal(t, tc.wantRecovery, recovery)
		})
	}
}

func TestPluginFailureTextKeepsTheRunsFlags(t *testing.T) {
	// The fallback after a plugin failure must not drop --experimental or
	// --output json, and must name only the failed agent.
	t.Setenv("PATH", t.TempDir())
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	item := agentPlanItem{agent: claude, delivery: deliveryPlugin, scope: agentScopeUser}
	inv := installInvocation{scope: installer.ScopeGlobal, agents: "claude-code,codex", experimental: true, jsonOutput: true}

	_, recovery := pluginFailureText(t.Context(), inv, item, &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: "not found"})
	assert.Equal(t, "Install or update the Claude Code CLI so 'claude' is on PATH, then re-run 'databricks aitools install --scope=global --agents=claude-code --experimental --output json'. Or install skill files instead: databricks aitools install --scope=global --agents=claude-code --skills-only --experimental --output json", recovery)
}

func TestSkipRecoveryNamesOnlyTheAgentInTheGlobalScope(t *testing.T) {
	cursor := agents.ByName(agents.NameCursor)
	inv := installInvocation{scope: installer.ScopeProject, agents: "claude-code,cursor", skillsOnly: true, experimental: true}
	assert.Equal(t, "To set up Cursor in the global scope, run: databricks aitools install --scope=global --agents=cursor --skills-only --experimental", skipRecovery(inv, agentPlanItem{agent: cursor, delivery: deliverySkip}))

	inv = installInvocation{scope: installer.ScopeProject, agents: "codex", jsonOutput: true}
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")
	assert.Equal(t, "To set up Codex CLI in the global scope, run: databricks aitools install --scope=global --agents=codex --output json", skipRecovery(inv, agentPlanItem{agent: codex, delivery: deliverySkip}))
}

func TestBuildInstallOutputIncludesRecovery(t *testing.T) {
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")
	ctx := cmdio.MockDiscard(t.Context())
	inv := installInvocation{scope: installer.ScopeGlobal, agents: agents.NameCodex, jsonOutput: true}
	out := buildInstallOutput(ctx, inv, installer.ScopeGlobal, []agentOutcome{{
		agent:    codex,
		delivery: deliveryPlugin,
		status:   outcomeFailed,
		message:  "codex: cli-not-on-path",
		recovery: "Install the Codex CLI",
	}}, &installer.StateError{Path: filepath.FromSlash("/s/.state.json"), Err: errors.New("failed to parse state file")})

	require.Len(t, out.Agents, 1)
	assert.Equal(t, "Install the Codex CLI", out.Agents[0].Recovery)
	assert.Equal(t, "failed to parse state file", out.Error, "the message stays the raw cause")
	assert.Contains(t, out.Recovery, "/s/.state.json")
	assert.Contains(t, out.Recovery, "re-run 'databricks aitools install --scope=global --agents=codex --output json'", "a JSON run is told to re-run in JSON")
}

func TestPluginFailureTextCLIWithoutPluginCommand(t *testing.T) {
	fakeBinsOnPath(t, "claude")
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	err := &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: "unknown command 'plugin'"}

	reason, recovery := pluginFailureText(t.Context(), installInvocation{scope: installer.ScopeGlobal}, agentPlanItem{agent: claude}, err)
	assert.Equal(t, "'claude plugin' is not available (unknown command 'plugin')", reason)
	// The CLI is already on PATH, so the step asks for a version with plugins.
	assert.Equal(t, "Update the Claude Code CLI to a version with the 'claude plugin' command, then re-run 'databricks aitools install --scope=global --agents=claude-code'. Or install skill files instead: databricks aitools install --scope=global --agents=claude-code --skills-only", recovery)
}

func TestRecoveryCommandsKeepTheProjectScope(t *testing.T) {
	// A project-scoped run must not be told to run a command that defaults to
	// the global scope or to every detected agent.
	t.Setenv("PATH", t.TempDir())
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	item := agentPlanItem{agent: claude, delivery: deliveryPlugin, scope: agentScopeProject}
	inv := installInvocation{scope: installer.ScopeProject}
	const want = "databricks aitools install --scope=project --agents=claude-code"

	notOnPath := &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: "not found"}
	_, recovery := pluginFailureText(t.Context(), inv, item, notOnPath)
	assert.Equal(t, "Install or update the Claude Code CLI so 'claude' is on PATH, then re-run '"+want+"'. Or install skill files instead: "+want+" --skills-only", recovery)
	assert.Contains(t, skillsFallbackText(inv, item), "run '"+want+"'.")
	assert.Contains(t, pluginKeptText(inv, item), "run '"+want+"'.")

	timedOut := &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonCLINotOnPath, Detail: "timed out after 20s", TimedOut: true}
	_, recovery = pluginFailureText(t.Context(), inv, item, timedOut)
	assert.Equal(t, "Re-run '"+want+"' (an agent CLI's first start can be slow), or install skill files instead: "+want+" --skills-only", recovery)

	failed := &installer.BlockedError{Agent: claude.Name, Reason: installer.ReasonInstallFailed, Detail: "boom"}
	_, recovery = pluginFailureText(t.Context(), inv, item, failed)
	assert.Contains(t, recovery, "'claude plugin install databricks@databricks-agent-skills --scope project'")
	assert.Contains(t, recovery, want+" --skills-only")
}

func TestRecoveryRepeatsTheRunsFlags(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "")
	inv := installInvocation{scope: installer.ScopeProject, agents: "cursor", skillsOnly: true, skills: "a", experimental: true}
	got := installRecovery(t.Context(), inv, &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusBadGateway, Err: errors.New("HTTP 502")})
	assert.Contains(t, got, "re-run 'databricks aitools install --scope=project --agents=cursor --skills-only --skills=a --experimental'.")
}

func TestPromptRecoveryBuildsOnTheChoicesMade(t *testing.T) {
	// The scope was chosen at the prompt; the agent picker failed.
	inv := installInvocation{scope: installer.ScopeProject, skillsOnly: true}
	assert.Equal(t, "To install without prompts, name the scope and agents, for example: databricks aitools install --scope=project --skills-only --agents=<name>", installRecovery(t.Context(), inv, errNoAgentsSelected))

	// The scope prompt failed; the agents were named by flag.
	inv = installInvocation{agents: agents.NameCursor}
	assert.Equal(t, "To install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=cursor", installRecovery(t.Context(), inv, huh.ErrUserAborted))
}

func TestRenderInstallCommandQuotesUserInput(t *testing.T) {
	tests := []struct {
		goos string
		inv  installInvocation
		want string
	}{
		{"linux", installInvocation{path: "/tmp/out"}, "databricks aitools install --path=/tmp/out"},
		{"darwin", installInvocation{path: "/tmp/my dir"}, "databricks aitools install --path='/tmp/my dir'"},
		{"linux", installInvocation{path: "/tmp/it's"}, `databricks aitools install --path='/tmp/it'\''s'`},
		// Single quotes keep every POSIX shell character literal.
		{"linux", installInvocation{path: "/tmp/$(id)"}, "databricks aitools install --path='/tmp/$(id)'"},
		{"linux", installInvocation{path: "/tmp/a`id`!%\"b"}, "databricks aitools install --path='/tmp/a`id`!%\"b'"},
		// A Windows path is printed with forward slashes: Windows accepts them,
		// and Git Bash would drop backslashes. A trailing one needs no doubling.
		{"windows", installInvocation{path: `C:\Users\me\out`}, `databricks aitools install --path=C:/Users/me/out`},
		{"windows", installInvocation{path: `C:\Users\me\my out`}, `databricks aitools install --path="C:/Users/me/my out"`},
		{"windows", installInvocation{path: `C:\Users\me\my out\`}, `databricks aitools install --path="C:/Users/me/my out/"`},

		// A skill name with a space stays one argument; normalized as install
		// parses the list, so "a, b" is one value too.
		{"linux", installInvocation{skillsOnly: true, skills: "databricks-jobs"}, "databricks aitools install --skills-only --skills=databricks-jobs"},
		{"linux", installInvocation{skillsOnly: true, skills: "data science, b"}, "databricks aitools install --skills-only --skills='data science,b'"},
		{"linux", installInvocation{skillsOnly: true, skills: "$(id)"}, "databricks aitools install --skills-only --skills='$(id)'"},
		{"windows", installInvocation{skillsOnly: true, skills: "data science, b"}, `databricks aitools install --skills-only --skills="data science,b"`},

		// A list of agents is one argument too: PowerShell splits at a bare comma.
		{"windows", installInvocation{scope: installer.ScopeGlobal, agents: "claude-code,cursor", skillsOnly: true, skills: "a,b"}, `databricks aitools install --scope=global --agents="claude-code,cursor" --skills-only --skills="a,b"`},
		{"linux", installInvocation{scope: installer.ScopeGlobal, agents: "claude-code,cursor"}, "databricks aitools install --scope=global --agents='claude-code,cursor'"},
		{"windows", installInvocation{scope: installer.ScopeGlobal, agents: "cursor"}, "databricks aitools install --scope=global --agents=cursor"},

		{"linux", installInvocation{scope: installer.ScopeGlobal, agents: "cursor", jsonOutput: true}, "databricks aitools install --scope=global --agents=cursor --output json"},
	}
	for _, tc := range tests {
		t.Run(tc.goos+" "+tc.want, func(t *testing.T) {
			got, ok := renderInstallCommand(tc.goos, tc.inv)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRerunPhraseDelimitsACommandThatContainsAQuote(t *testing.T) {
	assert.Equal(t, "'databricks aitools install --path=/tmp/out'", rerunPhrase("linux", installInvocation{path: "/tmp/out"}))
	assert.Equal(t, "`databricks aitools install --path='my skills'`", rerunPhrase("linux", installInvocation{path: "my skills"}))
	assert.Equal(t, `'databricks aitools install --path="my skills"'`, rerunPhrase("windows", installInvocation{path: "my skills"}))
}

func TestRenderInstallCommandRejectsValuesThatDoNotPasteLiterally(t *testing.T) {
	tests := []struct {
		goos string
		inv  installInvocation
	}{
		// A line break ends the pasted command and runs what follows, in any shell.
		{"linux", installInvocation{skillsOnly: true, skills: "a\nrm -rf ~"}},
		{"darwin", installInvocation{path: "/tmp/a\r\nb"}},
		{"windows", installInvocation{skillsOnly: true, skills: "a\r\ndel *"}},
		// PowerShell expands $ and ` inside double quotes, cmd.exe %VAR% and
		// !VAR!, and a double quote ends the string.
		{"windows", installInvocation{path: `C:\$cache\skills`}},
		{"windows", installInvocation{skillsOnly: true, skills: "$(id)"}},
		{"windows", installInvocation{path: "C:\\a`b"}},
		{"windows", installInvocation{path: `C:\%TEMP%\skills`}},
		{"windows", installInvocation{path: `C:\!TEMP!\skills`}},
		{"windows", installInvocation{path: `C:\a"b`}},
	}
	for _, tc := range tests {
		t.Run(tc.goos+" "+tc.inv.path+tc.inv.skills, func(t *testing.T) {
			_, ok := renderInstallCommand(tc.goos, tc.inv)
			assert.False(t, ok)
			assert.Equal(t, "your original install command with --debug", rerunPhrase(tc.goos, tc.inv, "--debug"))
		})
	}
}

func TestRecoveryAsksToRepeatTheCommandForAValueThatCannotBePrinted(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "")
	inv := installInvocation{scope: installer.ScopeGlobal, skillsOnly: true, skills: "a\nb"}
	err := &installer.FetchError{Stage: installer.FetchStageManifest, Err: dnsError}
	assert.Equal(t, "Could not download Databricks skills from raw.githubusercontent.com. Check that this machine can reach it (set HTTPS_PROXY if you use a proxy), then re-run your original install command.", installRecovery(t.Context(), inv, err))
	assert.Equal(t, "Re-run your original install command with --debug to see details. If it keeps failing, report it at https://github.com/databricks/cli/issues.", installRecovery(t.Context(), inv, errors.New("boom")))

	cursor := agents.ByName(agents.NameCursor)
	assert.Equal(t, "To set up Cursor in the global scope, run: your original install command with --scope=global --agents=cursor", skipRecovery(inv, agentPlanItem{agent: cursor, delivery: deliverySkip}))
}

func TestStateRecoveryDropsNamedSkills(t *testing.T) {
	serr := &installer.StateError{Path: filepath.FromSlash("/home/u/.databricks/aitools/skills/.state.json"), Err: errors.New("failed to parse state file")}
	inv := installInvocation{scope: installer.ScopeGlobal, agents: "cursor", skillsOnly: true, skills: "databricks-jobs"}
	assert.Equal(t, "Move the install state file /home/u/.databricks/aitools/skills/.state.json aside (for example, rename it to .state.json.bak), then re-run 'databricks aitools install --scope=global --agents=cursor --skills-only' to reinstall all skills (without the state file, an install of named skills in the global scope is refused). The state file also tracked the other agents set up in this scope; re-run their install too, for example 'databricks aitools install --scope=global'.", installRecovery(t.Context(), inv, serr))
}

func TestStateRecoveryKeepsNamedSkillsInTheProjectScope(t *testing.T) {
	// The installer refuses a targeted install without state only in the
	// global scope, so a project-scope re-run keeps --skills.
	serr := &installer.StateError{Path: filepath.FromSlash("/repo/.databricks/aitools/skills/.state.json"), Err: errors.New("failed to parse state file")}
	inv := installInvocation{scope: installer.ScopeProject, agents: "cursor", skillsOnly: true, skills: "databricks-jobs"}
	assert.Equal(t, "Move the install state file /repo/.databricks/aitools/skills/.state.json aside (for example, rename it to .state.json.bak), then re-run 'databricks aitools install --scope=project --agents=cursor --skills-only --skills=databricks-jobs'. The new state file records only the named skills, so the project's other skills stay untracked by update and uninstall; reinstall them too with 'databricks aitools install --scope=project --agents=cursor --skills-only'. The state file also tracked the other agents set up in this scope; re-run their install too, for example 'databricks aitools install --scope=project'.", installRecovery(t.Context(), inv, serr))
}

func TestRecoveryNamesAnOverrideWithoutAScheme(t *testing.T) {
	// The CLI fetched what the variable says; there is no other host to name.
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "mirror.internal/skills")
	got := installRecovery(t.Context(), installInvocation{}, &installer.FetchError{Stage: installer.FetchStageManifest, Err: &url.Error{Op: "Get", URL: "mirror.internal/skills/v1/manifest.json", Err: dnsError}})
	assert.Contains(t, got, "Could not download Databricks skills from mirror.internal/skills.")
}

func TestRefusalDoesNotHideAJoinedFailure(t *testing.T) {
	// A refused skills install joined to a permission error on the state file:
	// the category is PERMISSION_DENIED, so the ownership step is printed.
	refused := &installer.RefusedError{Err: errors.New("refusing to downgrade skills from 0.3.0 to 0.2.0")}
	denied := fmt.Errorf("failed to save state: %w", &fs.PathError{Op: "open", Path: filepath.FromSlash("/home/u/.databricks/aitools/skills/.state.json"), Err: fs.ErrPermission})
	got := installRecovery(t.Context(), installInvocation{}, errors.Join(refused, denied))
	assert.Equal(t, "Check that you own /home/u/.databricks/aitools/skills/.state.json and can write to it (an earlier run with sudo can leave it owned by root), then re-run 'databricks aitools install'.", got)
}

func TestFilesystemRecoveryIsNotABugReport(t *testing.T) {
	full := fmt.Errorf("failed to write skill: %w", &fs.PathError{Op: "write", Path: filepath.FromSlash("/home/u/.claude/skills/x/SKILL.md"), Err: syscall.ENOSPC})
	assert.Equal(t, "Check that /home/u/.claude/skills/x/SKILL.md is on a writable filesystem with free space, then re-run 'databricks aitools install'.", installRecovery(t.Context(), installInvocation{}, full))
}

func TestPinnedRefRecoveryMentionsTheVariableOnlyWhenSet(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v9.9.9")
	err := &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusNotFound, Err: fmt.Errorf("skills manifest at x@v9.9.9: %w", clicompat.ErrNotFound)}
	assert.Equal(t, "No skills release exists at the version DATABRICKS_SKILLS_REF names. Unset it, or set it to a release tag from https://github.com/databricks/databricks-agent-skills/releases, then re-run 'databricks aitools install'.", installRecovery(t.Context(), installInvocation{}, err))
}
