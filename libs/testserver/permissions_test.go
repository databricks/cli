package testserver_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionsObjectTypes(t *testing.T) {
	// Include non-bundle objects: deriving bundle routes must preserve the other APIs.
	expected := map[string]string{
		"cluster-policies":        "cluster-policy",
		"instance-pools":          "instance-pool",
		"clusters":                "cluster",
		"pipelines":               "pipelines",
		"jobs":                    "job",
		"notebooks":               "notebook",
		"directories":             "directory",
		"files":                   "file",
		"repos":                   "repo",
		"authorization":           "tokens",
		"sql/warehouses":          "warehouses",
		"dbsql-dashboards":        "dashboard",
		"sql/alerts":              "alert",
		"sql/queries":             "query",
		"dashboards":              "dashboard",
		"genie":                   "genie-space",
		"experiments":             "mlflowExperiment",
		"registered-models":       "registered-model",
		"serving-endpoints":       "serving-endpoint",
		"vector-search-endpoints": "vector-search-endpoints",
		"apps":                    "apps",
		"database-instances":      "database-instances",
		"database-projects":       "database-projects",
		"alertsv2":                "alertv2",
	}
	for requestType, responseType := range expected {
		t.Run(requestType, func(t *testing.T) {
			workspace := &testserver.FakeWorkspace{Permissions: make(map[string]iam.ObjectPermissions)}
			vars := map[string]string{"object_type": requestType, "object_id": "123"}
			if prefix, objectType, ok := strings.Cut(requestType, "/"); ok {
				vars["prefix"], vars["object_type"] = prefix, objectType
			}
			level := "CAN_MANAGE"
			switch requestType {
			case "jobs", "pipelines":
				level = "IS_OWNER"
			case "cluster-policies":
				level = "CAN_USE"
			}
			response := workspace.SetPermissions(testserver.Request{
				Vars: vars,
				Body: fmt.Appendf(nil, `{"access_control_list":[{"user_name":"alice","permission_level":%q}]}`, level),
			}).(testserver.Response)
			require.Zero(t, response.StatusCode)
			body, ok := response.Body.(iam.ObjectPermissions)
			require.True(t, ok)
			assert.Equal(t, responseType, body.ObjectType)
			assert.Equal(t, "/"+requestType+"/123", body.ObjectId)
		})
	}
}
