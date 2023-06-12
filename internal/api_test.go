package internal

import (
	"encoding/json"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/databricks/cli/cmd/api"
)

func TestAcxApiGet(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	stdout, _ := RequireSuccessfulRun(t, "api", "get", "/api/2.0/preview/scim/v2/Me")

	// Deserialize SCIM API response.
	var out map[string]any
	err := json.Unmarshal(stdout.Bytes(), &out)
	require.NoError(t, err)

	// Assert that the output somewhat makes sense for the SCIM API.
	assert.Equal(t, true, out["active"])
	assert.NotNil(t, out["id"])
}

func TestAcxApiPost(t *testing.T) {
	env := GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)
	if env == "gcp" {
		t.Skip("DBFS REST API is disabled on gcp")
	}

	dbfsPath := path.Join("/tmp/databricks/integration", RandomName("api-post"))
	requestPath := writeFile(t, "body.json", fmt.Sprintf(`{
		"path": "%s"
	}`, dbfsPath))

	// Post to mkdir
	{
		RequireSuccessfulRun(t, "api", "post", "--json=@"+requestPath, "/api/2.0/dbfs/mkdirs")
	}

	// Post to delete
	{
		RequireSuccessfulRun(t, "api", "post", "--json=@"+requestPath, "/api/2.0/dbfs/delete")
	}
}
