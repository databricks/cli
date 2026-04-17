package hostmetadata_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allowlist maps repo-relative paths (forward slashes) to a short reason the
// site doesn't need hostmetadata.Attach. When you add a new entry, write a
// specific reason — "no resolution" is too vague; say "SaveToProfile: write-only".
var allowlist = map[string]string{
	"cmd/auth/auth.go":                       "CanonicalHostName only (URL munging)",
	"cmd/auth/resolve.go":                    "CanonicalHostName only",
	"cmd/auth/logout.go":                     "CanonicalHostName only",
	"cmd/auth/token.go":                      "SaveToProfile: write-only",
	"cmd/configure/configure.go":             "SaveToProfile: write-only",
	"libs/databrickscfg/profile/profile.go":  "CanonicalHostName only",
	"libs/databrickscfg/profile/profiler.go": "CanonicalHostName only",
	"libs/testproxy/server.go":               "test helper, no real auth",
	"acceptance/internal/prepare_server.go":  "acceptance test infrastructure",
	"libs/env/loader.go":                     "doc comment only, no struct construction",
	// Task 6 deliberately skipped these two sites:
	//   cmd/auth/login.go:setHostAndAccountId (used for HostType() pattern matching only)
	//   cmd/root/auth.go:~290 (cfg reassigned from already-resolved client)
	// Both are in files that ALSO contain Attach calls, so they don't appear
	// in this allowlist — the file-level "has Attach" check covers them.
}

// constructionPattern matches both `config.Config{` and `databricks.Config{`
// struct literals — the two forms we construct in this repo.
var constructionPattern = regexp.MustCompile(`\b(?:config|databricks)\.Config\{`)

func TestConfigConstructionSitesHaveAttach(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var offenders []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip: .git, vendor, .claude (worktrees), acceptance test output dirs.
			name := d.Name()
			if name == ".git" || name == "vendor" || name == ".claude" || name == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files — we only want production code.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		// Allowlist check: if the file is explicitly allowlisted, skip.
		if _, ok := allowlist[relSlash]; ok {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(src)

		if !constructionPattern.MatchString(content) {
			return nil
		}
		if strings.Contains(content, "hostmetadata.Attach(") {
			return nil
		}

		offenders = append(offenders, relSlash)
		return nil
	})
	require.NoError(t, err)

	assert.Empty(t, offenders,
		"the following files construct *config.Config but do not call hostmetadata.Attach. "+
			"Either add `hostmetadata.Attach(cfg)` before the first resolve, "+
			"or add the file to the allowlist in %s with a specific reason.",
		"libs/hostmetadata/injection_guardrail_test.go")
}

// findRepoRoot walks up from the test's working directory until it finds go.mod.
func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod walking up from " + dir)
		}
		dir = parent
	}
}
