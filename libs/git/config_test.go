package git

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	// Taken from https://git-scm.com/docs/git-config#_example.
	raw := `
# Core variables
[core]
	; Don't trust file modes
	filemode = false

# Our diff algorithm
[diff]
	external = /usr/local/bin/diff-wrapper
	renames = true

[branch "devel"]
	remote = origin
	merge = refs/heads/devel

# Proxy settings
[core]
	gitProxy="ssh" for "kernel.org"
	gitProxy=default-proxy ; for the rest

[include]
	path = /path/to/foo.inc ; include by absolute path
	path = foo.inc ; find "foo.inc" relative to the current file
	path = ~/foo.inc ; find "foo.inc" in your $HOME directory

; include if $GIT_DIR is /path/to/foo/.git
[includeIf "gitdir:/path/to/foo/.git"]
	path = /path/to/foo.inc

; include for all repositories inside /path/to/group
[includeIf "gitdir:/path/to/group/"]
	path = /path/to/foo.inc

; include for all repositories inside $HOME/to/group
[includeIf "gitdir:~/to/group/"]
	path = /path/to/foo.inc

; relative paths are always relative to the including
; file (if the condition is true); their location is not
; affected by the condition
[includeIf "gitdir:/path/to/group/"]
	path = foo.inc

; include only if we are in a worktree where foo-branch is
; currently checked out
[includeIf "onbranch:foo-branch"]
	path = foo.inc

; include only if a remote with the given URL exists (note
; that such a URL may be provided later in a file or in a
; file read after this file is read, as seen in this example)
[includeIf "hasconfig:remote.*.url:https://example.com/**"]
	path = foo.inc
[remote "origin"]
	url = https://example.com/git
`

	c, err := newConfig()
	require.NoError(t, err)

	err = c.load(bytes.NewBufferString(raw))
	require.NoError(t, err)

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	assert.Equal(t, "false", c.variables["core.filemode"])
	assert.Equal(t, "origin", c.variables["branch.devel.remote"])

	// Verify that ~/ expands to the user's home directory.
	assert.Equal(t, filepath.Join(home, "foo.inc"), c.variables["include.path"])
}

func TestCoreExcludesFile(t *testing.T) {
	config, err := globalGitConfig()
	require.NoError(t, err)
	path, err := config.coreExcludesFile()
	require.NoError(t, err)
	t.Log(path)
}

type testCoreExcludesHelper struct {
	*testing.T

	home          string
	xdgConfigHome string
}

func (h *testCoreExcludesHelper) initialize(t *testing.T) {
	h.T = t

	// Create temporary $HOME directory.
	h.home = t.TempDir()
	t.Setenv("HOME", h.home)
	t.Setenv("USERPROFILE", h.home)

	// Create temporary $XDG_CONFIG_HOME directory.
	h.xdgConfigHome = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", h.xdgConfigHome)

	xdgConfigHomeGit := filepath.Join(h.xdgConfigHome, "git")
	err := os.MkdirAll(xdgConfigHomeGit, 0o755)
	require.NoError(t, err)
}

func (h *testCoreExcludesHelper) coreExcludesFile() (string, error) {
	config, err := globalGitConfig()
	require.NoError(h.T, err)
	return config.coreExcludesFile()
}

func (h *testCoreExcludesHelper) writeConfig(path, contents string) {
	err := os.WriteFile(path, []byte(contents), 0o644)
	require.NoError(h, err)
}

func TestCoreExcludesFileDefaultWithXdgConfigHome(t *testing.T) {
	h := &testCoreExcludesHelper{}
	h.initialize(t)

	path, err := h.coreExcludesFile()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(h.xdgConfigHome, "git/ignore"), path)
}

func TestCoreExcludesFileDefaultWithoutXdgConfigHome(t *testing.T) {
	h := &testCoreExcludesHelper{}
	h.initialize(t)
	h.Setenv("XDG_CONFIG_HOME", "")

	path, err := h.coreExcludesFile()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(h.home, ".config/git/ignore"), path)
}

func TestCoreExcludesFileSetInXdgConfigHomeGitConfig(t *testing.T) {
	h := &testCoreExcludesHelper{}
	h.initialize(t)
	h.writeConfig(filepath.Join(h.xdgConfigHome, "git/config"), `
[core]
excludesFile = ~/foo
	`)

	path, err := h.coreExcludesFile()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(h.home, "foo"), path)
}

func TestCoreExcludesFileSetInHomeGitConfig(t *testing.T) {
	h := &testCoreExcludesHelper{}
	h.initialize(t)
	h.writeConfig(filepath.Join(h.home, ".gitconfig"), `
[core]
excludesFile = ~/foo
	`)

	path, err := h.coreExcludesFile()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(h.home, "foo"), path)
}

func TestCoreExcludesFileSetInBoth(t *testing.T) {
	h := &testCoreExcludesHelper{}
	h.initialize(t)
	h.writeConfig(filepath.Join(h.xdgConfigHome, ".gitconfig"), `
[core]
excludesFile = ~/foo1
	`)
	h.writeConfig(filepath.Join(h.home, ".gitconfig"), `
[core]
excludesFile = ~/foo2
	`)

	path, err := h.coreExcludesFile()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(h.home, "foo2"), path)
}
