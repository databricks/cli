package agents

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopInstall(context.Context) error { return nil }

func TestRecommendSkillsInstallSkipsWhenSkillsExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	// Skills must be in canonical location to be detected.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, CanonicalSkillsDir, "databricks"), 0o755))

	agentDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	err := RecommendSkillsInstall(ctx, noopInstall)
	assert.NoError(t, err)
	assert.Empty(t, hintKinds(t, ctx))
}

func TestRecommendSkillsInstallSkipsWhenPluginInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	writeManifest(t, tmpDir, `{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.6"}]}}`)

	origRegistry := Registry
	Registry = []*Agent{agentWithConfigDir(tmpDir)}
	defer func() { Registry = origRegistry }()

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	err := RecommendSkillsInstall(ctx, noopInstall)
	assert.NoError(t, err)
	assert.Empty(t, stderr.String())
	assert.Empty(t, hintKinds(t, ctx))
}

func TestRecommendSkillsInstallSkipsWhenNoAgents(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	origRegistry := Registry
	Registry = []*Agent{}
	defer func() { Registry = origRegistry }()

	ctx := cmdio.MockDiscard(t.Context())
	err := RecommendSkillsInstall(ctx, noopInstall)
	assert.NoError(t, err)
}

func TestRecommendSkillsInstallNonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	err := RecommendSkillsInstall(ctx, noopInstall)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "databricks aitools install")
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindAppsInitTip}, hintKinds(t, ctx))
}

func TestRecommendSkillsInstallInteractiveDecline(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx, testIO := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer testIO.Done()
	ctx = telemetry.WithNewLogger(ctx)

	// Drain stderr so the prompt write doesn't block.
	go func() { _, _ = io.Copy(io.Discard, testIO.Stderr) }()

	errc := make(chan error, 1)
	go func() {
		errc <- RecommendSkillsInstall(ctx, noopInstall)
	}()

	_, err := testIO.Stdin.WriteString("n\n")
	require.NoError(t, err)
	require.NoError(t, testIO.Stdin.Flush())

	assert.NoError(t, <-errc)
	// The prompt is logged when shown, so a declined prompt still counts.
	assert.Equal(t, []protos.AitoolsHintKind{protos.AitoolsHintKindAppsInitPrompt}, hintKinds(t, ctx))
}

func TestMaybeVersionCheckTip(t *testing.T) {
	noPlugins := func(context.Context) []string { return nil }
	recordedCodexPlugin := func(context.Context) []string { return []string{"codex_0.2.6"} }

	for _, tc := range []struct {
		name             string
		agentEnv         []string
		agentDirs        []string
		globalSkills     bool
		claudePlugin     bool
		projectSkills    bool
		agentSkills      bool
		fromSubdir       bool
		onDBR            bool
		installedPlugins func(context.Context) []string
		wantTip          bool
	}{
		{name: "agent without tooling", agentDirs: []string{".codex"}, installedPlugins: noPlugins, wantTip: true},
		{name: "no agent detected", installedPlugins: noPlugins},
		{name: "plugin in agent manifest", agentDirs: []string{".claude"}, claudePlugin: true, installedPlugins: noPlugins},
		{name: "plugin recorded in install state", agentDirs: []string{".codex"}, installedPlugins: recordedCodexPlugin},
		{name: "global skills", agentDirs: []string{".kiro"}, globalSkills: true, installedPlugins: noPlugins},
		{name: "project skills", agentDirs: []string{".kiro"}, projectSkills: true, installedPlugins: noPlugins},
		{name: "project skills at the repository root, run from a subdirectory", agentDirs: []string{".kiro"}, projectSkills: true, fromSubdir: true, installedPlugins: noPlugins},
		{name: "skills only in the agent's own skills dir", agentDirs: []string{".cursor"}, agentSkills: true, installedPlugins: noPlugins},
		{name: "databricks runtime", agentDirs: []string{".codex"}, onDBR: true, installedPlugins: noPlugins},
		{name: "claude code caller is left to MaybeHint", agentEnv: []string{"CLAUDECODE", "1"}, agentDirs: []string{".claude"}, installedPlugins: noPlugins},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetAgentEnv(t, append([]string{"HOME", home, "USERPROFILE", home}, tc.agentEnv...)...)
			project := t.TempDir()
			if tc.fromSubdir {
				// The walk up to the repository root stops at the nearest .git.
				require.NoError(t, os.Mkdir(filepath.Join(project, ".git"), 0o755))
				subdir := filepath.Join(project, "src")
				require.NoError(t, os.Mkdir(subdir, 0o755))
				t.Chdir(subdir)
			} else {
				t.Chdir(project)
			}
			for _, dir := range tc.agentDirs {
				require.NoError(t, os.MkdirAll(filepath.Join(home, dir), 0o755))
			}
			if tc.claudePlugin {
				writePluginManifest(t, home, `{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.6"}]}}`)
			}
			if tc.globalSkills {
				require.NoError(t, os.MkdirAll(filepath.Join(home, CanonicalSkillsDir, "databricks"), 0o755))
			}
			if tc.projectSkills {
				require.NoError(t, os.MkdirAll(filepath.Join(project, ".kiro", "skills", "databricks-apps"), 0o755))
			}
			if tc.agentSkills {
				require.NoError(t, os.MkdirAll(filepath.Join(home, ".cursor", "skills", "databricks-apps"), 0o755))
			}

			ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
			ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: tc.onDBR})
			MaybeVersionCheckTip(ctx, tc.installedPlugins)
			if tc.wantTip {
				assert.Equal(t, skillsTipText+"\n", stderr.String())
			} else {
				assert.Empty(t, stderr.String())
			}
		})
	}
}
