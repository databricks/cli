package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func StartServer(t *testing.T) *testserver.Server {
	server := testserver.New(t)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func AddHandlers(server *testserver.Server) {
	server.Handle("/api/2.0/policies/clusters/list", func(r *http.Request) (any, error) {
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
		}, nil
	})

	server.Handle("/api/2.0/instance-pools/list", func(r *http.Request) (any, error) {
		return compute.ListInstancePools{
			InstancePools: []compute.InstancePoolAndStats{
				{
					InstancePoolName: "some-test-instance-pool",
					InstancePoolId:   "1234",
				},
			},
		}, nil
	})

	server.Handle("/api/2.1/clusters/list", func(r *http.Request) (any, error) {
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
		}, nil
	})

	server.Handle("/api/2.0/preview/scim/v2/Me", func(r *http.Request) (any, error) {
		return iam.User{
			UserName: "tester@databricks.com",
		}, nil
	})

	server.Handle("/api/2.0/workspace/get-status", func(r *http.Request) (any, error) {
		return workspace.ObjectInfo{
			ObjectId:   1001,
			ObjectType: "DIRECTORY",
			Path:       "",
			ResourceId: "1001",
		}, nil
	})

	server.Handle("/api/2.1/unity-catalog/current-metastore-assignment", func(r *http.Request) (any, error) {
		return catalog.MetastoreAssignment{
			DefaultCatalogName: "main",
		}, nil
	})

	server.Handle("/api/2.0/permissions/directories/1001", func(r *http.Request) (any, error) {
		return workspace.WorkspaceObjectPermissions{
			ObjectId:   "1001",
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
		}, nil
	})
}
