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
			ChangedFiles []struct {
				AnyGlobToAllFiles []string `yaml:"any-glob-to-all-files"`
			} `yaml:"changed-files"`
		} `yaml:"test-only-fix"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &config))
	glob := config.TestOnlyFix[0].ChangedFiles[0].AnyGlobToAllFiles[0]

	// Directory entries look like "libs/testserver/**". Entries such as
	// "**/*_test.go" match by file name and are skipped.
	var dirs []string
	for alt := range strings.SplitSeq(strings.Trim(glob, "{}"), ",") {
		dir, ok := strings.CutSuffix(alt, "/**")
		if ok && !strings.Contains(dir, "*") {
			dirs = append(dirs, dir)
		}
	}
	require.NotEmpty(t, dirs)

	// Release targets from .goreleaser.yaml; some files build only on some of them.
	for _, target := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			goos, goarch, _ := strings.Cut(target, "/")
			// go test does not see the sources go list reads, so a cached local pass can be
			// stale. CI reruns it because a fresh checkout changes the config file's mtime.
			cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
			cmd.Dir = "../.."
			cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
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
		})
	}
}
