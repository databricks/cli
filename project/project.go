package project

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databricks/databricks-sdk-go/service/scim"
)

// Current CLI application state - fixure out
var Current inner

type inner struct {
	mu   sync.Mutex
	once sync.Once

	project *Project
	wsc     *workspaces.WorkspacesClient
	me      *scim.User
}

func (i *inner) init() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.once.Do(func() {
		client := &common.DatabricksClient{}
		i.wsc = workspaces.New()
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

func (i *inner) Project() *Project {
	i.init()
	return i.project
}

// Make sure to initialize the workspaces client on project init
func (i *inner) WorkspacesClient() *workspaces.WorkspacesClient {
	i.init()
	return i.wsc
}

// We can replace this with go sdk once https://github.com/databricks/databricks-sdk-go/issues/56 is fixed
func (i *inner) Me() *scim.User {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.me != nil {
		return i.me
	}
	me, err := i.wsc.CurrentUser.Me(context.Background())
	// me, err := scim.NewUsersAPI(context.Background(), i.Client()).Me()
	if err != nil {
		panic(err)
	}
	i.me = me
	return me
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
	wsc *workspaces.WorkspacesClient,
	clusterName string,
) (clusterId string, err error) {
	clusterId = ""
	clustersList, err := wsc.Clusters.List(ctx, clusters.ListRequest{})
	if err != nil {
		return
	}
	for _, cluster := range clustersList.Clusters {
		if cluster.ClusterName == clusterName {
			clusterId = cluster.ClusterId
			return
		}
	}
	err = fmt.Errorf("could not find cluster with name: %s", clusterName)
	return
}

// Old version of getting development cluster details with isolation implemented.
// Kept just for reference. Remove once isolation is implemented properly
/*
func (i *inner) DevelopmentCluster(ctx context.Context) (cluster clusters.ClusterInfo, err error) {
	api := clusters.NewClustersAPI(ctx, i.Client()) // TODO: rewrite with normal SDK
	if i.project.DevCluster == nil {
		i.project.DevCluster = &clusters.Cluster{}
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
	return api.GetOrCreateRunningCluster(dc.ClusterName, *dc)
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
	return exec.Execute(cluster.ClusterID, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) common.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}
*/

// TODO: Add safe access to i.project and i.project.DevCluster that throws errors if
// the fields are not defined properly
func (i *inner) GetDevelopmentClusterId(ctx context.Context) (clusterId string, err error) {
	i.init()
	clusterId = i.project.DevCluster.ClusterId
	clusterName := i.project.DevCluster.ClusterName
	if clusterId != "" {
		return
	} else if clusterName != "" {
		// Add workspaces client on init
		return getClusterIdFromClusterName(ctx, i.wsc, clusterName)
	} else {
		// TODO: Add the project config file location used to error message
		err = fmt.Errorf("please define either development cluster's cluster_id or cluster_name in your project config")
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
	return Current.wsc.Commands.Execute(ctx, clusterId, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) commands.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}
