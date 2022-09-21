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
	"github.com/spf13/cobra"
)

type project struct {
	mu sync.Mutex

	root string

	config *Config
	wsc    *workspaces.WorkspacesClient
	me     *scim.User
}

// Configure is used as a PreRunE function for all commands that
// require a project to be configured. If a project could successfully
// be found and loaded, it is set on the command's context object.
func Configure(cmd *cobra.Command, args []string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}

	ctx, err := Initialize(cmd.Context(), root)
	if err != nil {
		return err
	}

	cmd.SetContext(ctx)
	return nil
}

// Placeholder to use as unique key in context.Context.
var projectKey int

// Initialize loads a project configuration given a root.
// It stores the project on a new context.
// The project is available through the `Get()` function.
func Initialize(ctx context.Context, root string) (context.Context, error) {
	config, err := loadProjectConf(root)
	if err != nil {
		return nil, err
	}

	p := project{
		root:   root,
		config: &config,
	}

	if config.Profile == "" {
		// Bricks config doesn't define the profile to use, so go sdk will figure
		// out the auth credentials based on the enviroment.
		// eg. DATABRICKS_CONFIG_PROFILE can be used to select which profile to use
		// DATABRICKS_HOST and DATABRICKS_TOKEN can be used to set the workspace auth creds
		p.wsc = workspaces.New()
	} else {
		p.wsc = workspaces.New(&databricks.Config{Profile: config.Profile})
	}

	return context.WithValue(ctx, &projectKey, &p), nil
}

// Get returns the project as configured on the context.
// It panics if it isn't configured.
func Get(ctx context.Context) *project {
	project, ok := ctx.Value(&projectKey).(*project)
	if !ok {
		panic(`context not configured with project`)
	}
	return project
}

// Make sure to initialize the workspaces client on project init
func (p *project) WorkspacesClient() *workspaces.WorkspacesClient {
	return p.wsc
}

func (p *project) Root() string {
	return p.root
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
	clusterId, err := Get(ctx).GetDevelopmentClusterId(ctx)
	if err != nil {
		return commands.CommandResults{
			ResultType: "error",
			Summary:    err.Error(),
		}
	}
	return Get(ctx).wsc.Commands.Execute(ctx, clusterId, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) commands.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}
