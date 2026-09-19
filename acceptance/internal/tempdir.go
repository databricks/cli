package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/git"
)

// RequireTempDirNotInGitRepo fails fast if os.TempDir() sits inside a git
// repository. A stray .git directory above the temp dir (e.g. an empty
// /tmp/.git) makes the CLI believe all temporary directories are git
// repositories, which breaks dozens of tests with baffling diffs.
// Catch it here instead of chasing them.
func RequireTempDirNotInGitRepo(t *testing.T) {
	for dir := os.TempDir(); ; {
		gitPath := filepath.Join(dir, git.GitDirectoryName)
		if _, err := os.Stat(gitPath); err == nil {
			t.Fatalf("temp dir %q is inside a git repository (found %q); the CLI would treat every temporary directory as a git repository - remove %q",
				os.TempDir(), gitPath, gitPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
