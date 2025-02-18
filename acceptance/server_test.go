package acceptance_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

var testUser = iam.User{
	Id:       "1000012345",
	UserName: "tester@databricks.com",
}

func AddHandlers(server *testserver.Server) {
	server.Handle("GET", "/api/2.0/policies/clusters/list", func(req testserver.Request) any {
		return compute.ListPoliciesResponse{
			Policies: []compute.Policy{
				{
					PolicyId: "5678",
					Name:     "wrong-cluster-policy",
				},
				{
					PolicyId: "9876",
					Name:     "some-test-cluster-policy",
				},
			},
		}
	})

	server.Handle("GET", "/api/2.0/instance-pools/list", func(req testserver.Request) any {
		return compute.ListInstancePools{
			InstancePools: []compute.InstancePoolAndStats{
				{
					InstancePoolName: "some-test-instance-pool",
					InstancePoolId:   "1234",
				},
			},
		}
	})

	server.Handle("GET", "/api/2.1/clusters/list", func(req testserver.Request) any {
		return compute.ListClustersResponse{
			Clusters: []compute.ClusterDetails{
				{
					ClusterName: "some-test-cluster",
					ClusterId:   "4321",
				},
				{
					ClusterName: "some-other-cluster",
					ClusterId:   "9876",
				},
			},
		}
	})

	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		return testserver.Response{
			Headers: map[string][]string{"X-Databricks-Org-Id": {"900800700600"}},
			Body:    testUser,
		}
	})

	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		return req.Workspace.WorkspaceGetStatus(path)
	})

	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req testserver.Request) any {
		var request workspace.Mkdirs
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		req.Workspace.WorkspaceMkdirs(request)
		return ""
	})

	server.Handle("GET", "/api/2.0/workspace/export", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		return req.Workspace.WorkspaceExport(path)
	})

	server.Handle("POST", "/api/2.0/workspace/delete", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		recursive := req.URL.Query().Get("recursive") == "true"
		req.Workspace.WorkspaceDelete(path, recursive)
		return ""
	})

	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path:.*}", func(req testserver.Request) any {
		path := req.Vars["path"]
		req.Workspace.WorkspaceFilesImportFile(path, req.Body)
		return ""
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req testserver.Request) any {
		return catalog.MetastoreAssignment{
			DefaultCatalogName: "main",
		}
	})

	server.Handle("GET", "/api/2.0/permissions/directories/{objectId}", func(req testserver.Request) any {
		objectId := req.Vars["objectId"]
		return workspace.WorkspaceObjectPermissions{
			ObjectId:   objectId,
			ObjectType: "DIRECTORY",
			AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "tester@databricks.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{
							PermissionLevel: "CAN_MANAGE",
						},
					},
				},
			},
		}
	})

	server.Handle("POST", "/api/2.1/jobs/create", func(req testserver.Request) any {
		var request jobs.CreateJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}

		return req.Workspace.JobsCreate(request)
	})

	server.Handle("GET", "/api/2.1/jobs/get", func(req testserver.Request) any {
		jobId := req.URL.Query().Get("job_id")
		return req.Workspace.JobsGet(jobId)
	})

	server.Handle("GET", "/api/2.1/jobs/list", func(req testserver.Request) any {
		return req.Workspace.JobsList()
	})

	server.Handle("GET", "/oidc/.well-known/oauth-authorization-server", func(_ testserver.Request) any {
		return map[string]string{
			"authorization_endpoint": server.URL + "oidc/v1/authorize",
			"token_endpoint":         server.URL + "/oidc/v1/token",
		}
	})

	server.Handle("POST", "/oidc/v1/token", func(_ testserver.Request) any {
		return map[string]string{
			"access_token": "oauth-token",
			"expires_in":   "3600",
			"scope":        "all-apis",
			"token_type":   "Bearer",
		}
	})
}
