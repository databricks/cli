package build

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

	// Full path of the package. For example: github.com/databricks/databricks-sdk-go@v0.47.1-0.20241002195128-6cecc224cbf7
	fullPath := fmt.Sprintf("%s@%s", modulePath, version)

	type goListResponse struct {
		Origin struct {
			Hash string
		}
	}

	// Using the go CLI query for the git hash corresponding to the databricks-sdk-go version
	cmd := exec.Command("go", "list", "-m", "-json", "-mod=readonly", fullPath)
	out, err := cmd.Output()
	require.NoError(t, err)
	parsedOutput := new(goListResponse)
	err = json.Unmarshal(out, parsedOutput)
	require.NoError(t, err)
	hash := parsedOutput.Origin.Hash
	require.NotEmpty(t, hash)

	// Read the OpenAPI SHA from the Go SDK.
	url := fmt.Sprintf("https://raw.githubusercontent.com/databricks/databricks-sdk-go/%s/.codegen/_openapi_sha", hash)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	sdkSha, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	cliSha, err := os.ReadFile("../../.codegen/_openapi_sha")
	require.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(string(cliSha)), strings.TrimSpace(string(sdkSha)), "please update the SDK version before generating the CLI")
}
