package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

func TestGetDetails(t *testing.T) {
	GetInfo()
}

// This test is stored in the internal package to avoid build errors since this
// test itself is run during the CLI generation process.
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

	// Clone the Databricks Go SDK to get the version of the OpenAPI spec used to
	// generate the SDK. We cannot use go mod vendor to download the SDK because
	// that command only downloads the necessary files and thus skips ".codegen".
	tmpDir := t.TempDir()
	err = git.Clone(context.Background(), "https://github.com/databricks/databricks-sdk-go.git", version, tmpDir)
	require.NoError(t, err)

	cliSha, err := os.ReadFile("../../.codegen/_openapi_sha")
	require.NoError(t, err)

	sdkSha, err := os.ReadFile(filepath.Join(tmpDir, ".codegen/_openapi_sha"))
	require.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(string(cliSha)), strings.TrimSpace(string(sdkSha)), "please update the SDK version before generating the CLI")
}
