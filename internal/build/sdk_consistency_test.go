package build

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

// This test ensures that the OpenAPI SHA the CLI is being generated from matches
// the OpenAPI SHA of the Go SDK version used in the CLI. We should always upgrade
// the Go SDK version before generating the CLI because downstream generated assets
// like the bundle schema depend on the Go SDK itself.
func TestConsistentDatabricksSdkVersion(t *testing.T) {
	// Read the go.mod file
	b, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)

	// Parse the go.mod file to get the databricks-sdk version
	modFile, err := modfile.Parse("../../go.mod", b, nil)
	require.NoError(t, err)

	modulePath := "github.com/databricks/databricks-sdk-go"
	var version string
	for _, r := range modFile.Require {
		if r.Mod.Path == modulePath {
			version = r.Mod.Version
		}
	}
	require.NotEmpty(t, version)

	// Read the OpenAPI SHA from the Go SDK at the version tag.
	url := fmt.Sprintf("https://raw.githubusercontent.com/databricks/databricks-sdk-go/%s/.codegen/_last_sha", version)
	resp, err := http.Get(url) //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	sdkSha, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	cliSha, err := os.ReadFile("../../.codegen/_openapi_sha")
	require.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(string(cliSha)), strings.TrimSpace(string(sdkSha)), "please update the SDK version before generating the CLI")
}

// This test ensures that the sdk-go modules the CLI requires are at their versions in the
// sdk-go release that climod/ was generated from, which climod/.codegen/_last_sha names by
// its Universe commit. The Go SDK update keeps them aligned, so that climod/ describes the
// sdk-go the CLI builds with.
func TestConsistentSdkGoVersions(t *testing.T) {
	marker, err := os.ReadFile("../../climod/.codegen/_last_sha")
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("climod/ hasn't been generated from an sdk-go release yet")
	}
	require.NoError(t, err)

	b, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)
	modFile, err := modfile.Parse("../../go.mod", b, nil)
	require.NoError(t, err)

	repo := t.TempDir()
	runGit(t, "", "clone", "--quiet", "--filter=blob:none", "--no-checkout", "https://github.com/databricks/sdk-go", repo)
	release := findSdkGoRelease(t, repo, strings.TrimSpace(string(marker)))

	for _, r := range modFile.Require {
		dir, ok := strings.CutPrefix(r.Mod.Path, "github.com/databricks/sdk-go/")
		if !ok {
			continue
		}
		// A release records each module's version, changed or not, as the first release
		// heading of the module's CHANGELOG.md.
		changelog := runGit(t, repo, "show", release+":"+dir+"/CHANGELOG.md")
		_, heading, found := strings.Cut(changelog, "## Release ")
		require.True(t, found, "%s/CHANGELOG.md has no release", dir)
		version := strings.Fields(heading)[0]
		assert.Equal(t, version, r.Mod.Version, "%s must match the sdk-go release in climod/.codegen/_last_sha; please update them together", r.Mod.Path)
	}
}

// findSdkGoRelease returns the newest tagged commit on sdk-go's main branch whose
// .codegen/_last_sha is universeSHA. A release merges before it's tagged, so only a tagged
// commit is published.
func findSdkGoRelease(t *testing.T, repo, universeSHA string) string {
	tagged := strings.Fields(runGit(t, repo, "rev-list", "--no-walk", "--tags"))
	for commit := range strings.FieldsSeq(runGit(t, repo, "rev-list", "--first-parent", "HEAD")) {
		// Releases from before the marker existed don't have it.
		if !slices.Contains(tagged, commit) || strings.TrimSpace(runGit(t, repo, "ls-tree", "--name-only", commit, ".codegen/_last_sha")) == "" {
			continue
		}
		if strings.TrimSpace(runGit(t, repo, "show", commit+":.codegen/_last_sha")) == universeSHA {
			return commit
		}
	}
	require.FailNow(t, "no sdk-go release was cut from the Universe commit in climod/.codegen/_last_sha", universeSHA)
	return ""
}

func runGit(t *testing.T, dir string, args ...string) string {
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err, "git %s", strings.Join(args, " "))
	return string(out)
}
