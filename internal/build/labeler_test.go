package build

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// TestLabelerTestOnlyDirsNotInBinary fails when the CLI binary imports a package
// under a directory that .github/labeler-test-only.yml treats as test-only, since
// a PR changing that directory would then be labeled test-only-fix.
func TestLabelerTestOnlyDirsNotInBinary(t *testing.T) {
	raw, err := os.ReadFile("../../.github/labeler-test-only.yml")
	require.NoError(t, err)

	var config struct {
		TestOnlyFix []struct {
			ChangedFiles []map[string][]string `yaml:"changed-files"`
		} `yaml:"test-only-fix"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &config))

	// The labeler ORs extra rules and globs into the label, and this test reads only
	// one glob, so a config that grows another one must fail here until it is covered.
	require.Len(t, config.TestOnlyFix, 1)
	require.Len(t, config.TestOnlyFix[0].ChangedFiles, 1)
	match := config.TestOnlyFix[0].ChangedFiles[0]
	require.Len(t, match, 1)
	globs := match["any-glob-to-all-files"]
	require.Len(t, globs, 1)

	// Directory entries look like "libs/testserver/**". Entries such as
	// "**/*_test.go" match by file name and are skipped.
	var dirs []string
	for alt := range strings.SplitSeq(strings.Trim(globs[0], "{}"), ",") {
		dir, ok := strings.CutSuffix(alt, "/**")
		if ok && !strings.Contains(dir, "*") {
			dirs = append(dirs, dir)
		}
	}
	require.NotEmpty(t, dirs)

	// go list resolves build constraints for the host platform only. CI runs this test
	// on Linux, macOS and Windows, which covers the OS-specific files.
	// go test does not see the sources go list reads, so a cached local pass can be
	// stale. CI reruns it because a fresh checkout changes the config file's mtime.
	cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.Output()
	require.NoError(t, err)

	for pkg := range strings.FieldsSeq(string(out)) {
		rel, ok := strings.CutPrefix(pkg, "github.com/databricks/cli/")
		if !ok {
			continue
		}
		for _, dir := range dirs {
			assert.False(t, rel == dir || strings.HasPrefix(rel, dir+"/"),
				"CLI binary imports %s, but .github/labeler-test-only.yml lists %s/** as test-only", pkg, dir)
		}
	}
}
