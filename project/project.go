package project

import (
	"context"
	"fmt"
	"sync"

	// "github.com/databrickslabs/terraform-provider-databricks/clusters"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databrickslabs/terraform-provider-databricks/scim"
)

// Current CLI application state - fixure out
var Current inner

type inner struct {
	mu   sync.Mutex
	once sync.Once

	project *Project
	workspaceClient *workspaces.WorkspacesClient
	client  *common.DatabricksClient
	me      *scim.User
}

func (i *inner) init() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.once.Do(func() {
		client := &common.DatabricksClient{}
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

// Make sure to initialize the workspaces client on project init
func (i *inner) WorkspacesClient() *workspaces.WorkspacesClient {
	i.init()
	return i.workspaceClient
}

// We can replace this with go sdk once https://github.com/databricks/databricks-sdk-go/issues/56 is fixed
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

func getClusterIdFromClusterName(ctx context.Context,
	workspaceClient *workspaces.WorkspacesClient, 
	clusterName string,
) (clusterId string, err error) {
	clusterId = ""
	listClustersResponse, err := workspaceClient.Clusters.ListClusters(ctx,clusters.ListClustersRequest{})
	if err != nil {
		return
	}
	for _, cluster := range listClustersResponse.Clusters {
		if cluster.ClusterName == clusterName {
			clusterId = cluster.ClusterId
			return
		}
	}
	err = fmt.Errorf("could not find cluster with name: %s", clusterName)
	return
}


// TODO: Add safe access to i.project and i.project.DevCluster that throws errors if
// the fields are not defined properly
func (i *inner) GetDevelopmentClusterId(ctx context.Context) (clusterId string, err error) {
	i.init()
	clusterId = i.project.DevCluster.ClusterId
	clusterName := i.project.DevCluster.ClusterName
	if (clusterId != "") {
		return
	} else if (clusterName != "") {
		// Add workspaces client on init
		return getClusterIdFromClusterName(ctx, i.workspaceClient, clusterName)
	} else {
		// TODO: Add the project config file location used to error message
		err = fmt.Errorf("Please define either development cluster's cluster_id or cluster_name in your project config")
		return
	}
}

func runCommandOnDev(ctx context.Context, language, command string) commands.CommandResults {
	clusterId, err := Current.GetDevelopmentClusterId(ctx)
	if err != nil {
		return commands.CommandResults{
			ResultType: "error",
			Summary:    err.Error(),
		}
	}
	return Current.workspaceClient.Commands.Execute(ctx, clusterId, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) commands.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}
