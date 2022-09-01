package project

import (
	"context"
	"fmt"
	"sync"

	// "github.com/databrickslabs/terraform-provider-databricks/clusters"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/databrickslabs/terraform-provider-databricks/commands"
	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databrickslabs/terraform-provider-databricks/scim"
	"github.com/jinzhu/copier"
)

// Current CLI application state - fixure out
var Current inner

type inner struct {
	mu   sync.Mutex
	once sync.Once

	project *Project
	client  *common.DatabricksClient
	me      *scim.User
}

func (i *inner) init() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.once.Do(func() {
		client := &common.DatabricksClient{}
		client.WithCommandExecutor(func(
			ctx context.Context, c *common.DatabricksClient) common.CommandExecutor {
			return commands.NewCommandsAPI(ctx, c)
		})
		i.client = client
		prj, err := loadProjectConf()
		if err != nil {
			panic(err)
		}
		client.Profile = prj.Profile // Databricks CLI profile
		err = client.Configure()
		if err != nil {
			panic(err)
		}
		
		i.project = &prj
	})
}

func (i *inner) Client() *common.DatabricksClient {
	i.init()
	return i.client
}

func (i *inner) Project() *Project {
	i.init()
	return i.project
}

func (i *inner) Me() *scim.User {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.me != nil {
		return i.me
	}
	me, err := scim.NewUsersAPI(context.Background(), i.Client()).Me()
	if err != nil {
		panic(err)
	}
	i.me = &me
	return &me
}

func (i *inner) DeploymentIsolationPrefix() string {
	if i.project.Isolation == None {
		return i.project.Name
	}
	if i.project.Isolation == Soft {
		me := i.Me()
		return fmt.Sprintf("%s/%s", i.project.Name, me.UserName)
	}
	panic(fmt.Errorf("unknow project isolation: %s", i.project.Isolation))
}

func (i *inner) DevelopmentCluster(ctx context.Context) (cluster clusters.ClusterInfo, err error) {
	workspacesClient := workspaces.New()
	clustersApiClient := workspacesClient.Clusters.(*clusters.ClustersAPI)
	if i.project.DevCluster == nil {
		i.project.DevCluster = &clusters.ClusterInfo{}
	}
	dc := i.project.DevCluster
	if i.project.Isolation == Soft {
		if i.project.IsDevClusterJustReference() {
			err = fmt.Errorf("projects with soft isolation cannot have named clusters")
			return
		}
		dc.ClusterName = fmt.Sprintf("dev/%s", i.DeploymentIsolationPrefix())
	}
	if dc.ClusterName == "" {
		err = fmt.Errorf("please either pick `isolation: soft` or specify a shared cluster name")
		return
	}
	clusterCreateRequest := clusters.CreateClusterRequest{}
	err = copier.Copy(&clusterCreateRequest, dc)
	if err != nil {
		return
	}
	clusterCreateResponse, err :=  clustersApiClient.GetOrCreateRunningCluster(ctx,
		clusterCreateRequest.ClusterName,
		clusterCreateRequest,
	)
	// Remove this once https://github.com/databricks/databricks-sdk-go/issues/53 is fixed
	getClusterResponse, err := clustersApiClient.GetCluster(ctx,
		clusters.GetClusterRequest{
			ClusterId: clusterCreateResponse.ClusterId,
		},
	)
	if err != nil {
		return
	}
	err = copier.Copy(&cluster, getClusterResponse)
	if err != nil {
		return
	}
	return
}

func runCommandOnDev(ctx context.Context, language, command string) common.CommandResults {
	cluster, err := Current.DevelopmentCluster(ctx)
	exec := Current.Client().CommandExecutor(ctx)
	if err != nil {
		return common.CommandResults{
			ResultType: "error",
			Summary:    err.Error(),
		}
	}
	return exec.Execute(cluster.ClusterId, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) common.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}
