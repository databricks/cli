package localenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootPM reports a fixed workspace root (or error) from WorkspaceRoot.
type rootPM struct {
	fakePM
	root string
	err  error
}

func (f rootPM) WorkspaceRoot(context.Context, string) (string, error) { return f.root, f.err }

func memberLayout(t *testing.T) (root, member string) {
	root = t.TempDir()
	member = filepath.Join(root, "bundles", "a")
	require.NoError(t, os.MkdirAll(member, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, pyprojectFile), []byte("[tool.uv.workspace]\nmembers = [\"bundles/*\"]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(member, pyprojectFile), []byte("[project]\nname = \"a\"\n"), 0o644))
	return root, member
}

func TestDiscoverWorkspaceRootMember(t *testing.T) {
	root, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: root}}
	assert.Equal(t, root, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootStandalone(t *testing.T) {
	_, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: member}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootSymlinkedSameDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	_, member := memberLayout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(member, link))
	// uv prints the resolved path; ProjectDir is the symlinked one.
	p := &Pipeline{ProjectDir: link, PM: rootPM{root: member}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootSkipsGreenfield(t *testing.T) {
	root := t.TempDir()
	member := filepath.Join(root, "new")
	require.NoError(t, os.MkdirAll(member, 0o755))
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: root}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootUvFailure(t *testing.T) {
	_, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{err: errors.New("unrecognized subcommand 'workspace'")}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestWorkspaceDirArgs(t *testing.T) {
	assert.Equal(t, []string{"workspace", "dir"}, (&uvManager{}).workspaceDirArgs())
}
