package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

var TestMetastore = catalog.MetastoreAssignment{
	DefaultCatalogName: "hive_metastore",
	MetastoreId:        "120efa64-9b68-46ba-be38-f319458430d2",
	WorkspaceId:        470123456789500,
}

func AddDefaultHandlers(server *Server) {
	server.Handle("GET", "/api/2.0/policies/clusters/list", func(req Request) any {
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

	server.Handle("GET", "/api/2.0/instance-pools/list", func(req Request) any {
		return compute.ListInstancePools{
			InstancePools: []compute.InstancePoolAndStats{
				{
					InstancePoolName: "some-test-instance-pool",
					InstancePoolId:   "1234",
				},
			},
		}
	})

	server.Handle("GET", "/api/2.1/clusters/list", func(req Request) any {
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

	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req Request) any {
		return Response{
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
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
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
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
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
		err := json.Unmarshal(req.Body, &request)
		if err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		if request.Format != workspace.ImportFormatAuto {
			return Response{
				Body:       "internal error: The test server only supports auto format.",
				StatusCode: http.StatusInternalServerError,
			}
		}

		// The /workspace/import endpoint expects the content as base64 encoded string.
		// We need to decode it to get the actual content.
		decoded, err := base64.StdEncoding.DecodeString(request.Content)
		if err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		return req.Workspace.WorkspaceFilesImportFile(request.Path, decoded, request.Overwrite)
	})

	server.Handle("GET", "/api/2.0/workspace-files/{path:.*}", func(req Request) any {
		path := req.Vars["path"]
		return req.Workspace.WorkspaceFilesExportFile(path)
	})

	server.Handle("HEAD", "/api/2.0/fs/directories/{path:.*}", func(req Request) any {
		return Response{
			Body: "dir path: " + req.Vars["dir_path"],
		}
	})

	server.Handle("PUT", "/api/2.0/fs/files/{path:.*}", func(req Request) any {
		path := req.Vars["path"]
		overwrite := req.URL.Query().Get("overwrite") == "true"
		return req.Workspace.WorkspaceFilesImportFile(path, req.Body, overwrite)
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req Request) any {
		return TestMetastore
	})

	server.Handle("POST", "/api/2.2/jobs/create", func(req Request) any {
		return req.Workspace.JobsCreate(req)
	})

	server.Handle("POST", "/api/2.2/jobs/delete", func(req Request) any {
		var request jobs.DeleteJob
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("request parsing error: %s", err),
			}
		}
		return MapDelete(req.Workspace, req.Workspace.Jobs, request.JobId)
	})

	server.Handle("POST", "/api/2.2/jobs/reset", func(req Request) any {
		return req.Workspace.JobsReset(req)
	})

	server.Handle("GET", "/api/2.0/jobs/get", func(req Request) any {
		return req.Workspace.JobsGet(req)
	})

	server.Handle("GET", "/api/2.2/jobs/get", func(req Request) any {
		return req.Workspace.JobsGet(req)
	})

	server.Handle("GET", "/api/2.2/jobs/list", func(req Request) any {
		return req.Workspace.JobsList()
	})

	server.Handle("POST", "/api/2.2/jobs/run-now", func(req Request) any {
		return req.Workspace.JobsRunNow(req)
	})

	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req Request) any {
		return req.Workspace.JobsGetRun(req)
	})

	server.Handle("GET", "/api/2.2/jobs/runs/list", func(req Request) any {
		return MapList(req.Workspace, req.Workspace.JobRuns, "runs")
	})

	server.Handle("GET", "/oidc/.well-known/oauth-authorization-server", func(_ Request) any {
		return server.fakeOidc.OidcEndpoints()
	})

	server.Handle("GET", "/oidc/v1/authorize", func(req Request) any {
		return server.fakeOidc.OidcAuthorize(req)
	})

	server.Handle("POST", "/oidc/v1/token", func(req Request) any {
		return server.fakeOidc.OidcToken(req)
	})

	server.Handle("POST", "/telemetry-ext", func(_ Request) any {
		return map[string]any{
			"errors":          []string{},
			"numProtoSuccess": 1,
		}
	})

	// Dashboards:
	server.Handle("GET", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Dashboards, req.Vars["dashboard_id"])
	})
	server.Handle("POST", "/api/2.0/lakeview/dashboards", func(req Request) any {
		return req.Workspace.DashboardCreate(req)
	})
	server.Handle("POST", "/api/2.0/lakeview/dashboards/{dashboard_id}/published", func(req Request) any {
		return req.Workspace.DashboardPublish(req)
	})
	server.Handle("PATCH", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req Request) any {
		return req.Workspace.DashboardUpdate(req)
	})
	server.Handle("DELETE", "/api/2.0/lakeview/dashboards/{dashboard_id}", func(req Request) any {
		// Calling DELETE on a dashboard does not delete the record immediately. Instead, the dashboard
		// is marked as trashed and moved to the trash.
		return req.Workspace.DashboardTrash(req)
	})
	server.Handle("GET", "/api/2.0/lakeview/dashboards/{dashboard_id}/published", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.PublishedDashboards, req.Vars["dashboard_id"])
	})

	// Genie Spaces:
	server.Handle("GET", "/api/2.0/genie/spaces/{space_id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.GenieSpaces, req.Vars["space_id"])
	})
	server.Handle("POST", "/api/2.0/genie/spaces", func(req Request) any {
		return req.Workspace.GenieSpaceCreate(req)
	})
	server.Handle("PATCH", "/api/2.0/genie/spaces/{space_id}", func(req Request) any {
		return req.Workspace.GenieSpaceUpdate(req, req.Vars["space_id"])
	})
	server.Handle("DELETE", "/api/2.0/genie/spaces/{space_id}", func(req Request) any {
		return req.Workspace.GenieSpaceTrash(req.Vars["space_id"])
	})

	// Pipelines:

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return req.Workspace.PipelineGet(req.Vars["pipeline_id"])
	})

	server.Handle("POST", "/api/2.0/pipelines", func(req Request) any {
		return req.Workspace.PipelineCreate(req)
	})

	server.Handle("PUT", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return req.Workspace.PipelineUpdate(req, req.Vars["pipeline_id"])
	})

	server.Handle("DELETE", "/api/2.0/pipelines/{pipeline_id}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Pipelines, req.Vars["pipeline_id"])
	})

	server.Handle("POST", "/api/2.0/pipelines/{pipeline_id}/updates", func(req Request) any {
		return req.Workspace.PipelineStartUpdate(req.Vars["pipeline_id"])
	})

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/events", func(req Request) any {
		return req.Workspace.PipelineEvents(req.Vars["pipeline_id"])
	})

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/updates/{update_id}", func(req Request) any {
		return req.Workspace.PipelineGetUpdate(req.Vars["pipeline_id"], req.Vars["update_id"])
	})

	server.Handle("POST", "/api/2.0/pipelines/{pipeline_id}/stop", func(req Request) any {
		return req.Workspace.PipelineStop(req.Vars["pipeline_id"])
	})

	// Quality monitors:

	server.Handle("GET", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], false)
	})

	server.Handle("PUT", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], true)
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})

	// Apps:

	server.Handle("GET", "/api/2.0/apps/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Apps, req.Vars["name"])
	})

	server.Handle("POST", "/api/2.0/apps", func(req Request) any {
		return req.Workspace.AppsUpsert(req, "")
	})

	server.Handle("PATCH", "/api/2.0/apps/{name}", func(req Request) any {
		return req.Workspace.AppsUpsert(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/apps/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Apps, req.Vars["name"])
	})

	// Schemas:

	server.Handle("GET", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Schemas, req.Vars["full_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/schemas", func(req Request) any {
		return req.Workspace.SchemasCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return req.Workspace.SchemasUpdate(req, req.Vars["full_name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Schemas, req.Vars["full_name"])
	})

	// Grants:

	server.Handle("PATCH", "/api/2.1/unity-catalog/permissions/{securable_type}/{full_name}", func(req Request) any {
		return req.Workspace.GrantsUpdate(req, req.Vars["securable_type"], req.Vars["full_name"])
	})

	server.Handle("GET", "/api/2.1/unity-catalog/permissions/{securable_type}/{full_name}", func(req Request) any {
		return req.Workspace.GrantsGet(req, req.Vars["securable_type"], req.Vars["full_name"])
	})

	// Catalogs:

	server.Handle("GET", "/api/2.1/unity-catalog/catalogs/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Catalogs, req.Vars["name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/catalogs", func(req Request) any {
		return req.Workspace.CatalogsCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/catalogs/{name}", func(req Request) any {
		return req.Workspace.CatalogsUpdate(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/catalogs/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Catalogs, req.Vars["name"])
	})

	// Registered Models:

	server.Handle("GET", "/api/2.1/unity-catalog/models/{full_name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.RegisteredModels, req.Vars["full_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/models", func(req Request) any {
		return req.Workspace.RegisteredModelsCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/models/{full_name}", func(req Request) any {
		return req.Workspace.RegisteredModelsUpdate(req, req.Vars["full_name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/models/{full_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.RegisteredModels, req.Vars["full_name"])
	})

	// Volumes:

	server.Handle("GET", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Volumes, req.Vars["full_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/volumes", func(req Request) any {
		return req.Workspace.VolumesCreate(req)
	})

	// Repos:

	server.Handle("POST", "/api/2.0/repos", func(req Request) any {
		return req.Workspace.ReposCreate(req)
	})

	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Repos, req.Vars["repo_id"])
	})

	server.Handle("PATCH", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return req.Workspace.ReposUpdate(req)
	})

	server.Handle("DELETE", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return req.Workspace.ReposDelete(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return req.Workspace.VolumesUpdate(req, req.Vars["full_name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Volumes, req.Vars["full_name"])
	})

	// SQL Warehouses:

	server.Handle("GET", "/api/2.0/sql/warehouses/{warehouse_id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.SqlWarehouses, req.Vars["warehouse_id"])
	})

	server.Handle("GET", "/api/2.0/sql/warehouses", func(req Request) any {
		return req.Workspace.SqlWarehousesList(req)
	})

	server.Handle("POST", "/api/2.0/sql/warehouses", func(req Request) any {
		return req.Workspace.SqlWarehousesUpsert(req, "")
	})

	server.Handle("POST", "/api/2.0/sql/warehouses/{warehouse_id}/edit", func(req Request) any {
		return req.Workspace.SqlWarehousesUpsert(req, req.Vars["warehouse_id"])
	})

	server.Handle("DELETE", "/api/2.0/sql/warehouses/{warehouse_id}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.SqlWarehouses, req.Vars["warehouse_id"])
	})

	server.Handle("GET", "/api/2.0/preview/sql/data_sources", func(req Request) any {
		return req.Workspace.SqlDataSourcesList(req)
	})

	// Alerts v2:
	server.Handle("GET", "/api/2.0/alerts/{id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Alerts, req.Vars["id"])
	})

	server.Handle("GET", "/api/2.0/alerts", func(req Request) any {
		return MapList(req.Workspace, req.Workspace.Alerts, "alerts")
	})

	server.Handle("POST", "/api/2.0/alerts", func(req Request) any {
		return req.Workspace.AlertsUpsert(req, "")
	})

	server.Handle("PATCH", "/api/2.0/alerts/{id}", func(req Request) any {
		return req.Workspace.AlertsUpsert(req, req.Vars["id"])
	})

	server.Handle("DELETE", "/api/2.0/alerts/{id}", func(req Request) any {
		return req.Workspace.AlertsDelete(req.Vars["id"])
	})

	// Secret Scopes:
	server.Handle("POST", "/api/2.0/secrets/scopes/create", func(req Request) any {
		return req.Workspace.SecretsCreateScope(req)
	})

	server.Handle("GET", "/api/2.0/secrets/scopes/list", func(req Request) any {
		return req.Workspace.SecretsListScopes(req)
	})

	server.Handle("POST", "/api/2.0/secrets/scopes/delete", func(req Request) any {
		return req.Workspace.SecretsDeleteScope(req)
	})

	// Secrets:
	server.Handle("POST", "/api/2.0/secrets/put", func(req Request) any {
		return req.Workspace.SecretsPut(req)
	})

	server.Handle("GET", "/api/2.0/secrets/get", func(req Request) any {
		return req.Workspace.SecretsGet(req)
	})

	// Secrets ACLs:
	server.Handle("GET", "/api/2.0/secrets/acls/get", func(req Request) any {
		return req.Workspace.SecretsAclsGet(req)
	})

	server.Handle("GET", "/api/2.0/secrets/acls/list", func(req Request) any {
		return req.Workspace.SecretsListAcls(req)
	})

	server.Handle("POST", "/api/2.0/secrets/acls/put", func(req Request) any {
		return req.Workspace.SecretsAclsPut(req)
	})

	server.Handle("POST", "/api/2.0/secrets/acls/delete", func(req Request) any {
		return req.Workspace.SecretsAclsDelete(req)
	})
	// Database Instances:

	server.Handle("POST", "/api/2.0/database/instances", func(req Request) any {
		return req.Workspace.DatabaseInstanceCreate(req)
	})

	server.Handle("GET", "/api/2.0/database/instances/", func(req Request) any {
		return MapList(req.Workspace, req.Workspace.DatabaseInstances, "database_instances")
	})

	server.Handle("GET", "/api/2.0/database/instances/{name}", func(req Request) any {
		return DatabaseInstanceMapGet(req.Workspace, req.Workspace.DatabaseInstances, req.Vars["name"])
	})

	server.Handle("PATCH", "/api/2.0/database/instances/{name}", func(req Request) any {
		return req.Workspace.DatabaseInstanceUpdate(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/database/instances/{name}", func(req Request) any {
		return DatabaseInstanceMapDelete(req)
	})

	// Database Catalogs:

	server.Handle("POST", "/api/2.0/database/catalogs", func(req Request) any {
		return req.Workspace.DatabaseCatalogCreate(req)
	})

	server.Handle("GET", "/api/2.0/database/catalogs/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.DatabaseCatalogs, req.Vars["name"])
	})

	server.Handle("PATCH", "/api/2.0/database/catalogs/{name}", func(req Request) any {
		return req.Workspace.DatabaseCatalogUpdate(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/database/catalogs/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.DatabaseCatalogs, req.Vars["name"])
	})

	// Synced Database Tables:

	server.Handle("POST", "/api/2.0/database/synced_tables", func(req Request) any {
		return req.Workspace.SyncedDatabaseTableCreate(req)
	})

	server.Handle("PATCH", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return req.Workspace.SyncedDatabaseTableUpdate(req, req.Vars["name"])
	})

	server.Handle("GET", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.SyncedDatabaseTables, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.SyncedDatabaseTables, req.Vars["name"])
	})

	// Clusters:
	server.Handle("POST", "/api/2.1/clusters/resize", func(req Request) any {
		return req.Workspace.ClustersResize(req)
	})

	server.Handle("POST", "/api/2.1/clusters/edit", func(req Request) any {
		return req.Workspace.ClustersEdit(req)
	})

	server.Handle("GET", "/api/2.1/clusters/get", func(req Request) any {
		clusterId := req.URL.Query().Get("cluster_id")
		return req.Workspace.ClustersGet(req, clusterId)
	})

	server.Handle("POST", "/api/2.1/clusters/create", func(req Request) any {
		return req.Workspace.ClustersCreate(req)
	})

	server.Handle("POST", "/api/2.1/clusters/start", func(req Request) any {
		return req.Workspace.ClustersStart(req)
	})

	server.Handle("POST", "/api/2.1/clusters/permanent-delete", func(req Request) any {
		return req.Workspace.ClustersPermanentDelete(req)
	})

	// MLflow Experiments:
	server.Handle("GET", "/api/2.0/mlflow/experiments/get", func(req Request) any {
		experimentId := req.URL.Query().Get("experiment_id")
		if experimentId == "" {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body:       map[string]string{"message": "experiment_id is required"},
			}
		}

		return MapGet(req.Workspace, req.Workspace.Experiments, experimentId)
	})

	server.Handle("POST", "/api/2.0/mlflow/experiments/create", func(req Request) any {
		return req.Workspace.ExperimentCreate(req)
	})

	server.Handle("POST", "/api/2.0/mlflow/experiments/update", func(req Request) any {
		return req.Workspace.ExperimentUpdate(req)
	})

	server.Handle("POST", "/api/2.0/mlflow/experiments/delete", func(req Request) any {
		return req.Workspace.ExperimentDelete(req)
	})

	// Model registry models.
	server.Handle("POST", "/api/2.0/mlflow/registered-models/create", func(req Request) any {
		return req.Workspace.ModelRegistryCreateModel(req)
	})

	server.Handle("GET", "/api/2.0/mlflow/databricks/registered-models/get", func(req Request) any {
		return req.Workspace.ModelRegistryGetModel(req)
	})

	server.Handle("PATCH", "/api/2.0/mlflow/registered-models/update", func(req Request) any {
		return req.Workspace.ModelRegistryUpdateModel(req)
	})

	server.Handle("DELETE", "/api/2.0/mlflow/registered-models/delete", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.ModelRegistryModels, req.URL.Query().Get("name"))
	})

	// Serving Endpoints:
	server.Handle("GET", "/api/2.0/serving-endpoints/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.ServingEndpoints, req.Vars["name"])
	})

	server.Handle("POST", "/api/2.0/serving-endpoints", func(req Request) any {
		return req.Workspace.ServingEndpointCreate(req)
	})

	server.Handle("PUT", "/api/2.0/serving-endpoints/{name}/config", func(req Request) any {
		return req.Workspace.ServingEndpointUpdate(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/serving-endpoints/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.ServingEndpoints, req.Vars["name"])
	})

	server.Handle("PUT", "/api/2.0/serving-endpoints/{name}/ai-gateway", func(req Request) any {
		return req.Workspace.ServingEndpointPutAiGateway(req, req.Vars["name"])
	})

	server.Handle("PATCH", "/api/2.0/serving-endpoints/{name}/notifications", func(req Request) any {
		return req.Workspace.ServingEndpointUpdateNotifications(req, req.Vars["name"])
	})

	server.Handle("PATCH", "/api/2.0/serving-endpoints/{name}/tags", func(req Request) any {
		return req.Workspace.ServingEndpointPatchTags(req, req.Vars["name"])
	})

	// Generic permissions endpoints
	server.Handle("GET", "/api/2.0/permissions/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.GetPermissions(req)
	})

	server.Handle("GET", "/api/2.0/permissions/{prefix}/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.GetPermissions(req)
	})

	server.Handle("PUT", "/api/2.0/permissions/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.SetPermissions(req)
	})

	server.Handle("PUT", "/api/2.0/permissions/{prefix}/{object_type}/{object_id}", func(req Request) any {
		return req.Workspace.SetPermissions(req)
	})
}
