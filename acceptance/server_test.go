package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/gorilla/mux"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func AddHandlers(server *testserver.Server) {
	server.Handle("GET", "/api/2.0/policies/clusters/list", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
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
		}, http.StatusOK
	})

	server.Handle("GET", "/api/2.0/instance-pools/list", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		return compute.ListInstancePools{
			InstancePools: []compute.InstancePoolAndStats{
				{
					InstancePoolName: "some-test-instance-pool",
					InstancePoolId:   "1234",
				},
			},
		}, http.StatusOK
	})

	server.Handle("GET", "/api/2.1/clusters/list", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
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
		}, http.StatusOK
	})

	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		return iam.User{
			Id:       "1000012345",
			UserName: "tester@databricks.com",
		}, http.StatusOK
	})

	server.Handle("GET", "/api/2.0/workspace/get-status", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		path := r.URL.Query().Get("path")

		return fakeWorkspace.WorkspaceGetStatus(path)
	})

	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		request := workspace.Mkdirs{}
		decoder := json.NewDecoder(r.Body)

		err := decoder.Decode(&request)
		if err != nil {
			return internalError(err)
		}

		return fakeWorkspace.WorkspaceMkdirs(request)
	})

	server.Handle("GET", "/api/2.0/workspace/export", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		path := r.URL.Query().Get("path")

		return fakeWorkspace.WorkspaceExport(path)
	})

	server.Handle("POST", "/api/2.0/workspace/delete", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		path := r.URL.Query().Get("path")
		recursiveStr := r.URL.Query().Get("recursive")
		var recursive bool

		if recursiveStr == "true" {
			recursive = true
		} else {
			recursive = false
		}

		return fakeWorkspace.WorkspaceDelete(path, recursive)
	})

	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path:.*}", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		vars := mux.Vars(r)
		path := vars["path"]

		body := new(bytes.Buffer)
		_, err := body.ReadFrom(r.Body)
		if err != nil {
			return internalError(err)
		}

		return fakeWorkspace.WorkspaceFilesImportFile(path, body.Bytes())
	})

	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		return catalog.MetastoreAssignment{
			DefaultCatalogName: "main",
		}, http.StatusOK
	})

	server.Handle("GET", "/api/2.0/permissions/directories/{objectId}", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		vars := mux.Vars(r)
		objectId := vars["objectId"]

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
		}, http.StatusOK
	})

	server.Handle("POST", "/api/2.1/jobs/create", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		request := jobs.CreateJob{}
		decoder := json.NewDecoder(r.Body)

		err := decoder.Decode(&request)
		if err != nil {
			return internalError(err)
		}

		return fakeWorkspace.JobsCreate(request)
	})

	server.Handle("GET", "/api/2.1/jobs/get", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		jobId := r.URL.Query().Get("job_id")

		return fakeWorkspace.JobsGet(jobId)
	})

	server.Handle("GET", "/api/2.1/jobs/list", func(fakeWorkspace *testserver.FakeWorkspace, r *http.Request) (any, int) {
		return fakeWorkspace.JobsList()
	})
}

func internalError(err error) (any, int) {
	return fmt.Errorf("internal error: %w", err), http.StatusInternalServerError
}
