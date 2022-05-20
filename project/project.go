package project

import (
	"context"
	"fmt"
	"sync"

	"github.com/databrickslabs/terraform-provider-databricks/clusters"
	"github.com/databrickslabs/terraform-provider-databricks/commands"
	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databrickslabs/terraform-provider-databricks/scim"
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
		me := i.Me()
		dc.ClusterName = fmt.Sprintf("dev/%s/%s", i.project.Name, me.UserName)
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
