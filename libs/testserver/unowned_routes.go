package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Default routes of APIs that have no file of their own in this package. They
// answer with fixed data or delegate to state shared by several APIs.

func init() {
	registerRoutes("unowned_routes", addUnownedRoutes)
}

func addUnownedRoutes(server *Server) {
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req Request) any {
		return Response{
			Headers: map[string][]string{"X-Databricks-Org-Id": {strconv.Itoa(TestWorkspaceID)}},
			Body:    req.Workspace.MeUser(req.Token),
		}
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req Request) any {
		return TestMetastore
	})

	server.Handle("GET", "/api/2.1/unity-catalog/metastore_summary", func(req Request) any {
		return catalog.GetMetastoreSummaryResponse{
			MetastoreId: TestMetastore.MetastoreId,
			Region:      "us-west-2",
		}
	})

	server.Handle("POST", "/telemetry-ext", func(req Request) any {
		var request struct {
			ProtoLogs []string `json:"protoLogs"`
		}
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return Response{
				Body:       fmt.Sprintf("invalid request body: %s", err),
				StatusCode: http.StatusBadRequest,
			}
		}
		// Accept every log in the request, as the real endpoint does.
		return map[string]any{
			"errors":          []string{},
			"numProtoSuccess": len(request.ProtoLogs),
		}
	})

	// Storage Credentials:

	server.Handle("GET", "/api/2.1/unity-catalog/storage-credentials", func(req Request) any {
		return catalog.ListStorageCredentialsResponse{
			StorageCredentials: []catalog.StorageCredentialInfo{
				{
					Name:        "some-test-credential",
					Id:          "1234",
					MetastoreId: "abcd",
				},
			},
		}
	})

	// Unity Catalog base endpoint (used for UC availability check):
	server.Handle("GET", "/api/2.1/unity-catalog", func(req Request) any {
		return map[string]any{
			"metastore_id": "test-metastore-id",
		}
	})
}
