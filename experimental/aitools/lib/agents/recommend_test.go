package agents

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopInstall(context.Context) error { return nil }

func TestRecommendSkillsInstallSkipsWhenSkillsExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Skills must be in canonical location to be detected.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, CanonicalSkillsDir, "databricks"), 0o755))

	agentDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx := cmdio.MockDiscard(context.Background())
	err := RecommendSkillsInstall(ctx, noopInstall)
	assert.NoError(t, err)
}

func TestRecommendSkillsInstallSkipsWhenNoAgents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origRegistry := Registry
	Registry = []Agent{}
	defer func() { Registry = origRegistry }()

	ctx := cmdio.MockDiscard(context.Background())
	err := RecommendSkillsInstall(ctx, noopInstall)
	assert.NoError(t, err)
}

func TestRecommendSkillsInstallNonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx, stderr := cmdio.NewTestContextWithStderr(context.Background())
	err := RecommendSkillsInstall(ctx, noopInstall)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "databricks experimental aitools skills install")
}

func TestRecommendSkillsInstallInteractiveDecline(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	ctx, testIO := cmdio.SetupTest(context.Background(), cmdio.TestOptions{PromptSupported: true})
	defer testIO.Done()

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
}
