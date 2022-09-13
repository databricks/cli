package fs

import (
	"context"
	"testing"

	"github.com/databricks/bricks/qa"
	"github.com/databricks/databricks-sdk-go/service/dbfs"
	"github.com/stretchr/testify/assert"
)

func TestLsCmd(t *testing.T) {
	fixtures := []qa.HTTPFixture{
		{
			Method:     "GET",
			RequestURI: "/api/2.0/dbfs/list?path=%2FMyDir",
			Response: dbfs.ListStatusResponse{
				Files: []dbfs.FileInfo{
					{Path: "/MyDir/hello.txt"},
					{Path: "/MyDir/world.txt"},
				},
			},
		},
	}
	wsc, server := qa.GeFixtureWorkspacesClient(t, fixtures)
	defer server.Close()

	_, err := wsc.Dbfs.ListByPath(context.TODO(), "/MyDir")
	assert.NoError(t, err)
}
