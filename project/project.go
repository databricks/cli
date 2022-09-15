package project

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/databricks-sdk-go/databricks"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

// Current CLI application state - fixure out
var Current project

type project struct {
	mu   sync.Mutex
	once sync.Once

	config *Config
	wsc    *workspaces.WorkspacesClient
	me     *scim.User
}

func (p *project) init() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.once.Do(func() {
		config, err := loadProjectConf()
		p.wsc = workspaces.New(&databricks.Config{Profile: config.Profile})
		if err != nil {
			panic(err)
		}
		if err != nil {
			panic(err)
		}
		p.config = &config
	})
}

// Make sure to initialize the workspaces client on project init
func (p *project) WorkspacesClient() *workspaces.WorkspacesClient {
	p.init()
	return p.wsc
}

func (p *project) Me() (*scim.User, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.me != nil {
		return p.me, nil
	}
	me, err := p.wsc.CurrentUser.Me(context.Background())
	if err != nil {
		return nil, err
	}
	p.me = me
	return me, nil
}

func (p *project) DeploymentIsolationPrefix() string {
	if p.config.Isolation == None {
		return p.config.Name
	}
	if p.config.Isolation == Soft {
		me, err := p.Me()
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%s/%s", p.config.Name, me.UserName)
	}
	panic(fmt.Errorf("unknow project isolation: %s", p.config.Isolation))
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
func (p *project) DevelopmentCluster(ctx context.Context) (cluster clusters.ClusterInfo, err error) {
	api := clusters.NewClustersAPI(ctx, p.Client()) // TODO: rewrite with normal SDK
	if p.project.DevCluster == nil {
		p.project.DevCluster = &clusters.Cluster{}
	}
	dc := p.project.DevCluster
	if p.project.Isolation == Soft {
		if p.project.IsDevClusterJustReference() {
			err = fmt.Errorf("projects with soft isolation cannot have named clusters")
			return
		}
		dc.ClusterName = fmt.Sprintf("dev/%s", p.DeploymentIsolationPrefix())
	}
	if dc.ClusterName == "" {
		err = fmt.Errorf("please either pick `isolation: soft` or specify a shared cluster name")
		return
	}
	return app.GetOrCreateRunningCluster(dc.ClusterName, *dc)
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

// TODO: Add safe access to p.project and p.project.DevCluster that throws errors if
// the fields are not defined properly
func (p *project) GetDevelopmentClusterId(ctx context.Context) (clusterId string, err error) {
	p.init()
	clusterId = p.config.DevCluster.ClusterId
	clusterName := p.config.DevCluster.ClusterName
	if clusterId != "" {
		return
	} else if clusterName != "" {
		// Add workspaces client on init
		return getClusterIdFromClusterName(ctx, p.wsc, clusterName)
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
