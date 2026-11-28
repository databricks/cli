package phases

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}

func freshnessRepositories(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.git")
	local := filepath.Join(dir, "local")
	writer := filepath.Join(dir, "writer")
	runGit(t, dir, "init", "--bare", "--initial-branch=main", remote)
	runGit(t, dir, "clone", remote, writer)
	runGit(t, writer, "config", "user.name", "Test")
	runGit(t, writer, "config", "user.email", "test@example.test")
	runGit(t, writer, "commit", "--allow-empty", "-m", "initial")
	runGit(t, writer, "push", "-u", "origin", "main")
	runGit(t, dir, "clone", remote, local)
	return local, writer
}

func freshnessContext(ctx context.Context, input string, interactive bool) (context.Context, *bytes.Buffer) {
	var stderr bytes.Buffer
	if interactive {
		ctx = cmdio.InContext(ctx, cmdio.NewTestIO(strings.NewReader(input), &bytes.Buffer{}, &stderr))
	} else {
		ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, strings.NewReader(input), &bytes.Buffer{}, &stderr, "", ""))
	}
	return logdiag.InitContext(ctx), &stderr
}

func TestCheckGitBranchFreshness(t *testing.T) {
	local, writer := freshnessRepositories(t)
	b := &bundle.Bundle{
		BundleRoot: vfs.MustNew(local),
		Config: config.Root{
			Experimental: &config.Experimental{CheckGitBranchFreshness: new(true)},
		},
	}

	ctx, output := freshnessContext(t.Context(), "", false)
	require.NoError(t, CheckGitBranchFreshness(ctx, b))
	assert.Empty(t, output.String())

	runGit(t, local, "config", "--replace-all", "remote.origin.fetch", "+refs/heads/other:refs/remotes/origin/other")
	runGit(t, writer, "commit", "--allow-empty", "-m", "one")
	runGit(t, writer, "commit", "--allow-empty", "-m", "two")
	runGit(t, writer, "push")

	tests := []struct {
		name        string
		input       string
		interactive bool
		autoApprove bool
		wantErr     string
	}{
		{name: "noninteractive", wantErr: "--auto-approve"},
		{name: "approved", input: "yes\n", interactive: true},
		{name: "declined", input: "no\n", interactive: true, wantErr: "deployment cancelled"},
		{name: "auto approved", autoApprove: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, output := freshnessContext(t.Context(), tt.input, tt.interactive)
			b.AutoApprove = tt.autoApprove
			err := CheckGitBranchFreshness(ctx, b)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Contains(t, output.String(), `Git branch "main" is 2 commit(s) behind its upstream "origin/main"`)
		})
	}

	b.Config.Experimental.CheckGitBranchFreshness = new(false)
	ctx, output = freshnessContext(t.Context(), "", false)
	require.NoError(t, CheckGitBranchFreshness(ctx, b))
	assert.Empty(t, output.String())
}

func TestCheckGitBranchFreshnessUnavailable(t *testing.T) {
	local, _ := freshnessRepositories(t)
	b := &bundle.Bundle{
		BundleRoot: vfs.MustNew(local),
		Config: config.Root{
			Experimental: &config.Experimental{CheckGitBranchFreshness: new(true)},
		},
	}

	runGit(t, local, "branch", "--unset-upstream")
	ctx, output := freshnessContext(t.Context(), "", false)
	require.NoError(t, CheckGitBranchFreshness(ctx, b))
	assert.Contains(t, output.String(), `cannot determine the upstream of branch "main"`)

	runGit(t, local, "branch", "--set-upstream-to=origin/main")
	runGit(t, local, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	ctx, output = freshnessContext(t.Context(), "", false)
	require.NoError(t, CheckGitBranchFreshness(ctx, b))
	assert.Contains(t, output.String(), "git fetch failed")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	ctx, output = freshnessContext(ctx, "", false)
	require.ErrorIs(t, CheckGitBranchFreshness(ctx, b), context.Canceled)
	assert.Empty(t, output.String())

	runGit(t, local, "checkout", "--detach")
	ctx, output = freshnessContext(t.Context(), "", false)
	require.NoError(t, CheckGitBranchFreshness(ctx, b))
	assert.Contains(t, output.String(), "cannot determine the current branch")
}

func TestCheckGitBranchFreshnessLocalUpstream(t *testing.T) {
	local, _ := freshnessRepositories(t)
	runGit(t, local, "config", "user.name", "Test")
	runGit(t, local, "config", "user.email", "test@example.test")
	runGit(t, local, "checkout", "-b", "other")
	runGit(t, local, "commit", "--allow-empty", "-m", "advance local branch")
	runGit(t, local, "checkout", "main")
	runGit(t, local, "branch", "--set-upstream-to=other")

	b := &bundle.Bundle{
		BundleRoot: vfs.MustNew(local),
		Config: config.Root{
			Experimental: &config.Experimental{CheckGitBranchFreshness: new(true)},
		},
	}
	ctx, output := freshnessContext(t.Context(), "", false)
	require.ErrorContains(t, CheckGitBranchFreshness(ctx, b), "--auto-approve")
	assert.Contains(t, output.String(), `Git branch "main" is 1 commit(s) behind its upstream "other"`)
}

func TestCheckGitBranchFreshnessRemoteURL(t *testing.T) {
	local, writer := freshnessRepositories(t)
	runGit(t, writer, "commit", "--allow-empty", "-m", "advance upstream")
	runGit(t, writer, "push")
	remote := filepath.Join(filepath.Dir(local), "remote.git")
	runGit(t, local, "config", "branch.main.remote", remote)

	b := &bundle.Bundle{
		BundleRoot: vfs.MustNew(local),
		Config: config.Root{
			Experimental: &config.Experimental{CheckGitBranchFreshness: new(true)},
		},
	}
	ctx, output := freshnessContext(t.Context(), "", false)
	require.ErrorContains(t, CheckGitBranchFreshness(ctx, b), "--auto-approve")
	assert.Contains(t, output.String(), `Git branch "main" is 1 commit(s) behind its upstream "main"`)
	assert.NotContains(t, output.String(), remote)
}
