package agents

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
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
