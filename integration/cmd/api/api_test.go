package api_test

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
)

func TestApiGet(t *testing.T) {
	stdout, _ := testcli.RequireSuccessfulRun(t, "api", "get", "/api/2.0/preview/scim/v2/Me")

	// Deserialize SCIM API response.
	var out map[string]any
	err := json.Unmarshal(stdout.Bytes(), &out)
	require.NoError(t, err)

	// Assert that the output somewhat makes sense for the SCIM API.
	assert.Equal(t, true, out["active"])
	assert.NotNil(t, out["id"])
}

func TestApiPost(t *testing.T) {
	if testutil.GetCloud(t) == testutil.GCP {
		t.Skip("DBFS REST API is disabled on gcp")
	}

	dbfsPath := path.Join("/tmp/databricks/integration", testutil.RandomName("api-post"))
	requestPath := filepath.Join(t.TempDir(), "body.json")
	testutil.WriteFile(t, requestPath, fmt.Sprintf(`{
		"path": "%s"
	}`, dbfsPath))

	// Post to mkdir
	{
		testcli.RequireSuccessfulRun(t, "api", "post", "--json=@"+requestPath, "/api/2.0/dbfs/mkdirs")
	}

	// Post to delete
	{
		testcli.RequireSuccessfulRun(t, "api", "post", "--json=@"+requestPath, "/api/2.0/dbfs/delete")
	}
}
