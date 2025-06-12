package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"

	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

var TestUser = iam.User{
	Id:       "1000012345",
	UserName: "tester@databricks.com",
}

var TestMetastore = catalog.MetastoreAssignment{
	DefaultCatalogName: "hive_metastore",
	MetastoreId:        "120efa64-9b68-46ba-be38-f319458430d2",
	WorkspaceId:        470123456789500,
}

func addDefaultHandlers(server *testserver.Server) {
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
			Body:    TestUser,
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

	server.Handle("GET", "/api/2.0/workspace-files/{path:.*}", func(req testserver.Request) any {
		path := req.Vars["path"]
		return req.Workspace.WorkspaceFilesExportFile(path)
	})

	server.Handle("HEAD", "/api/2.0/fs/directories/{path:.*}", func(req testserver.Request) any {
		return testserver.Response{
			Body: "dir path: " + req.Vars["dir_path"],
		}
	})

	server.Handle("PUT", "/api/2.0/fs/files/{path:.*}", func(req testserver.Request) any {
		path := req.Vars["path"]
		req.Workspace.WorkspaceFilesImportFile(path, req.Body)
		return ""
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req testserver.Request) any {
		return TestMetastore
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

	server.Handle("POST", "/api/2.2/jobs/create", func(req testserver.Request) any {
		var request jobs.CreateJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}

		return req.Workspace.JobsCreate(request)
	})

	server.Handle("POST", "/api/2.2/jobs/delete", func(req testserver.Request) any {
		var request jobs.DeleteJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}
		return testserver.MapDelete(req.Workspace, req.Workspace.Jobs, request.JobId)
	})

	server.Handle("POST", "/api/2.2/jobs/reset", func(req testserver.Request) any {
		var request jobs.ResetJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}

		return req.Workspace.JobsReset(request)
	})

	server.Handle("GET", "/api/2.2/jobs/get", func(req testserver.Request) any {
		jobId := req.URL.Query().Get("job_id")
		return req.Workspace.JobsGet(jobId)
	})

	server.Handle("GET", "/api/2.2/jobs/list", func(req testserver.Request) any {
		return req.Workspace.JobsList()
	})

	server.Handle("GET", "/api/2.2/jobs/list", func(req testserver.Request) any {
		return req.Workspace.JobsList()
	})

	server.Handle("POST", "/api/2.2/jobs/run-now", func(req testserver.Request) any {
		var request jobs.RunNow
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}

		return req.Workspace.JobsRunNow(request.JobId)
	})

	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		runId := req.URL.Query().Get("run_id")
		runIdInt, err := strconv.ParseInt(runId, 10, 64)
		if err != nil {
			return testserver.Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}

		return req.Workspace.JobsGetRun(runIdInt)
	})

	server.Handle("GET", "/api/2.2/jobs/runs/list", func(req testserver.Request) any {
		return testserver.MapList(req.Workspace, req.Workspace.JobRuns, "runs")
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

	server.Handle("POST", "/telemetry-ext", func(_ testserver.Request) any {
		return telemetry.ResponseBody{
			Errors:          []telemetry.LogError{},
			NumProtoSuccess: 1,
		}
	})

	// Dashboards:
	server.Handle("GET", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req testserver.Request) any {
		return testserver.MapGet(req.Workspace, req.Workspace.Dashboards, req.Vars["dashboard_id"])
	})
	server.Handle("POST", "/api/2.0/lakeview/dashboards", func(req testserver.Request) any {
		return req.Workspace.DashboardCreate(req)
	})
	server.Handle("POST", "/api/2.0/lakeview/dashboards/{dashboard_id}/published", func(req testserver.Request) any {
		return req.Workspace.DashboardPublish(req)
	})
	server.Handle("PATCH", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req testserver.Request) any {
		return req.Workspace.DashboardUpdate(req)
	})
	server.Handle("DELETE", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req testserver.Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Dashboards, req.Vars["dashboard_id"])
	})

	// Pipelines:

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		return req.Workspace.PipelineGet(req.Vars["pipeline_id"])
	})

	server.Handle("POST", "/api/2.0/pipelines", func(req testserver.Request) any {
		return req.Workspace.PipelineCreate(req)
	})

	server.Handle("PUT", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		return req.Workspace.PipelineUpdate(req, req.Vars["pipeline_id"])
	})

	server.Handle("DELETE", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Pipelines, req.Vars["pipeline_id"])
	})

	// Quality monitors:

	server.Handle("GET", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req testserver.Request) any {
		return testserver.MapGet(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req testserver.Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], false)
	})

	server.Handle("PUT", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req testserver.Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], true)
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req testserver.Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})

	// Apps:

	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		return testserver.MapGet(req.Workspace, req.Workspace.Apps, req.Vars["name"])
	})

	server.Handle("POST", "/api/2.0/apps", func(req testserver.Request) any {
		return req.Workspace.AppsUpsert(req, "")
	})

	server.Handle("PATCH", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		return req.Workspace.AppsUpsert(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Apps, req.Vars["name"])
	})

	// Schemas:

	server.Handle("GET", "/api/2.1/unity-catalog/schemas/{full_name}", func(req testserver.Request) any {
		return testserver.MapGet(req.Workspace, req.Workspace.Schemas, req.Vars["full_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/schemas", func(req testserver.Request) any {
		return req.Workspace.SchemasCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/schemas/{full_name}", func(req testserver.Request) any {
		return req.Workspace.SchemasUpdate(req, req.Vars["full_name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/schemas/{full_name}", func(req testserver.Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Schemas, req.Vars["full_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/volumes", func(req testserver.Request) any {
		return req.Workspace.VolumesCreate(req)
	})
}
