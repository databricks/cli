package build

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
	url := fmt.Sprintf("https://raw.githubusercontent.com/databricks/databricks-sdk-go/%s/.codegen/_openapi_sha", version)
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
