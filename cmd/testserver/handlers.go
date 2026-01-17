package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// toTestserverRequest converts our Request to testserver.Request for handlers that need it.
func toTestserverRequest(req Request) testserver.Request {
	return testserver.Request{
		Method:    req.Method,
		URL:       req.URL,
		Headers:   req.Headers,
		Body:      req.Body,
		Vars:      req.Vars,
		Workspace: req.Workspace,
		Context:   req.Context,
	}
}

func addDefaultHandlers(server *StandaloneServer) {
	server.Handle("GET", "/api/2.0/policies/clusters/list", func(req Request) any {
		return compute.ListPoliciesResponse{
			Policies: []compute.Policy{
				{PolicyId: "5678", Name: "wrong-cluster-policy"},
				{PolicyId: "9876", Name: "some-test-cluster-policy"},
			},
		}
	})

	server.Handle("GET", "/api/2.0/instance-pools/list", func(req Request) any {
		return compute.ListInstancePools{
			InstancePools: []compute.InstancePoolAndStats{
				{InstancePoolName: "some-test-instance-pool", InstancePoolId: "1234"},
			},
		}
	})

	server.Handle("GET", "/api/2.1/clusters/list", func(req Request) any {
		return compute.ListClustersResponse{
			Clusters: []compute.ClusterDetails{
				{ClusterName: "some-test-cluster", ClusterId: "4321"},
				{ClusterName: "some-other-cluster", ClusterId: "9876"},
			},
		}
	})

	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req Request) any {
		return testserver.Response{
			Headers: map[string][]string{"X-Databricks-Org-Id": {"900800700600"}},
			Body:    req.Workspace.CurrentUser(),
		}
	})

	server.Handle("GET", "/api/2.0/workspace/get-status", func(req Request) any {
		path := req.URL.Query().Get("path")
		return req.Workspace.WorkspaceGetStatus(path)
	})

	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req Request) any {
		var request workspace.Mkdirs
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		req.Workspace.WorkspaceMkdirs(request)
		return ""
	})

	server.Handle("GET", "/api/2.0/workspace/export", func(req Request) any {
		path := req.URL.Query().Get("path")
		return req.Workspace.WorkspaceExport(path)
	})

	server.Handle("POST", "/api/2.0/workspace/delete", func(req Request) any {
		var request workspace.Delete
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: 500}
		}
		req.Workspace.WorkspaceDelete(request.Path, request.Recursive)
		return ""
	})

	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path:.*}", func(req Request) any {
		path := req.Vars["path"]
		overwrite := req.URL.Query().Get("overwrite") == "true"
		return req.Workspace.WorkspaceFilesImportFile(path, req.Body, overwrite)
	})

	server.Handle("POST", "/api/2.0/workspace/import", func(req Request) any {
		var request workspace.Import
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		if request.Format != workspace.ImportFormatAuto {
			return testserver.Response{Body: "internal error: only auto format supported", StatusCode: http.StatusInternalServerError}
		}
		decoded, err := base64.StdEncoding.DecodeString(request.Content)
		if err != nil {
			return testserver.Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		return req.Workspace.WorkspaceFilesImportFile(request.Path, decoded, request.Overwrite)
	})

	server.Handle("GET", "/api/2.0/workspace-files/{path:.*}", func(req Request) any {
		return req.Workspace.WorkspaceFilesExportFile(req.Vars["path"])
	})

	server.Handle("HEAD", "/api/2.0/fs/directories/{path:.*}", func(req Request) any {
		return testserver.Response{Body: "dir path: " + req.Vars["path"]}
	})

	server.Handle("PUT", "/api/2.0/fs/files/{path:.*}", func(req Request) any {
		path := req.Vars["path"]
		overwrite := req.URL.Query().Get("overwrite") == "true"
		return req.Workspace.WorkspaceFilesImportFile(path, req.Body, overwrite)
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req Request) any {
		return testserver.TestMetastore
	})

	// Jobs
	server.Handle("POST", "/api/2.2/jobs/create", func(req Request) any {
		return req.Workspace.JobsCreate(toTestserverRequest(req))
	})

	server.Handle("POST", "/api/2.2/jobs/delete", func(req Request) any {
		var request jobs.DeleteJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return testserver.Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
		}
		return testserver.MapDelete(req.Workspace, req.Workspace.Jobs, request.JobId)
	})

	server.Handle("POST", "/api/2.2/jobs/reset", func(req Request) any {
		return req.Workspace.JobsReset(toTestserverRequest(req))
	})

	server.Handle("GET", "/api/2.0/jobs/get", func(req Request) any {
		return req.Workspace.JobsGet(toTestserverRequest(req))
	})

	server.Handle("GET", "/api/2.2/jobs/get", func(req Request) any {
		return req.Workspace.JobsGet(toTestserverRequest(req))
	})

	server.Handle("GET", "/api/2.2/jobs/list", func(req Request) any {
		return req.Workspace.JobsList()
	})

	server.Handle("POST", "/api/2.2/jobs/run-now", func(req Request) any {
		return req.Workspace.JobsRunNow(toTestserverRequest(req))
	})

	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req Request) any {
		return req.Workspace.JobsGetRun(toTestserverRequest(req))
	})

	server.Handle("GET", "/api/2.2/jobs/runs/list", func(req Request) any {
		return testserver.MapList(req.Workspace, req.Workspace.JobRuns, "runs")
	})

	// Pipelines
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return req.Workspace.PipelineGet(req.Vars["pipeline_id"])
	})

	server.Handle("POST", "/api/2.0/pipelines", func(req Request) any {
		return req.Workspace.PipelineCreate(toTestserverRequest(req))
	})

	server.Handle("PUT", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return req.Workspace.PipelineUpdate(toTestserverRequest(req), req.Vars["pipeline_id"])
	})

	server.Handle("DELETE", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return testserver.MapDelete(req.Workspace, req.Workspace.Pipelines, req.Vars["pipeline_id"])
	})

	// Tables
	server.Handle("GET", "/api/2.1/unity-catalog/tables/{full_name}", func(req Request) any {
		parts := strings.Split(req.Vars["full_name"], ".")
		if len(parts) != 3 {
			return testserver.Response{StatusCode: 400, Body: "Invalid table name"}
		}
		return testserver.Response{
			Body: catalog.TableInfo{
				CatalogName: parts[0],
				SchemaName:  parts[1],
				Name:        parts[2],
				FullName:    req.Vars["full_name"],
			},
		}
	})

	// Telemetry
	server.Handle("POST", "/telemetry-ext", func(req Request) any {
		return map[string]any{"errors": []string{}, "numProtoSuccess": 1}
	})

	// Permissions
	server.Handle("GET", "/api/2.0/permissions/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.GetPermissions(toTestserverRequest(req))
	})

	server.Handle("PUT", "/api/2.0/permissions/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.SetPermissions(toTestserverRequest(req))
	})
}
