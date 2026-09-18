package phases

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -qq hides the listing of resources about to be deleted only because --auto-approve
// makes it informational. Without --auto-approve the user is about to consent to the
// deletion and must see it, at any -q level. Prompting requires a TTY, which acceptance
// tests do not have (destroy errors out before this point), so it is covered here; the
// --auto-approve levels are covered by acceptance/bundle/deploy/quiet-levels.
func TestApprovalForDestroyQuietWhilePrompting(t *testing.T) {
	plan := &deployplan.Plan{
		Plan: map[string]*deployplan.PlanEntry{
			"resources.jobs.my_job": {ID: "1", Action: deployplan.Delete},
		},
	}

	stderr := &bytes.Buffer{}
	// "y" answers the prompt; cmdio panics reading from a nil stdin.
	ctx := cmdio.InContext(t.Context(),
		cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader("y\n"), io.Discard, stderr, "", ""))

	b := &bundle.Bundle{Quiet: bundle.QuietAll}
	b.Config.Workspace.RootPath = "/Workspace/Users/me/.bundle/x"

	approved, err := approvalForDestroy(ctx, b, plan, engine.EngineTerraform)
	require.NoError(t, err)
	assert.True(t, approved)

	assert.Contains(t, stderr.String(), "The following resources will be deleted:")
	assert.Contains(t, stderr.String(), "resources.jobs.my_job")
	assert.Contains(t, stderr.String(), b.Config.Workspace.RootPath)
}

func TestRemoveEmptyDirs(t *testing.T) {
	root := t.TempDir()
	// empty/nested/ is pruned bottom-up; keep/ holds a file, so it and root survive.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "empty", "nested"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "keep"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep", "state.json"), nil, 0o600))

	removed, err := removeEmptyDirs(root, 0)
	require.NoError(t, err)
	assert.False(t, removed)

	assert.NoDirExists(t, filepath.Join(root, "empty"))
	assert.FileExists(t, filepath.Join(root, "keep", "state.json"))
}

func TestRemoveEmptyDirsDepthLimit(t *testing.T) {
	// Entering past the cap trips the guard without building a pathological tree.
	_, err := removeEmptyDirs(t.TempDir(), maxStateDirDepth+1)
	assert.ErrorContains(t, err, "nesting exceeds")
}
