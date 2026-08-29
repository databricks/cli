package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Build a repository where main advances after a feature branch is cut.
	// This is the situation GetChangedFiles must handle correctly: the diff
	// should report only the feature branch's own changes, not files that moved
	// on main in the meantime. The repository is self-contained so the test does
	// not depend on the depth of the surrounding clone.
	t.Chdir(t.TempDir())

	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(file, contents string) {
		t.Helper()
		require.NoError(t, os.WriteFile(file, []byte(contents), 0o600))
		git("add", file)
		git("commit", "-q", "-m", file)
	}

	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.invalid")
	git("config", "user.name", "Test")

	commit("base.txt", "base") // common ancestor
	git("checkout", "-q", "-b", "feature")
	commit("feature.txt", "feature") // the change that belongs to the PR
	git("checkout", "-q", "main")
	commit("main-only.txt", "main-only") // main advances after the branch was cut

	// base is main's tip; head is the feature branch. A two-dot diff would also
	// list main-only.txt (absent on the feature side); the three-dot diff must
	// report only feature.txt.
	files, err := GetChangedFiles("feature", "main")
	require.NoError(t, err)
	assert.Equal(t, []string{"feature.txt"}, files)

	// Identical refs produce no changes.
	files, err = GetChangedFiles("feature", "feature")
	require.NoError(t, err)
	assert.Empty(t, files)

	// Invalid refs error.
	_, err = GetChangedFiles("invalid-ref-12345", "invalid-ref-67890")
	assert.Error(t, err)
}
