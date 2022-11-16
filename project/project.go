package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/bricks/git"
	"github.com/databricks/databricks-sdk-go/databricks"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/spf13/cobra"
)

const CacheDirName = ".databricks"

type project struct {
	mu sync.Mutex

	root string
	env  string

	config      *Config
	environment *Environment
	wsc         *workspaces.WorkspacesClient
	me          *scim.User
	fileSet     *git.FileSet
}

// Configure is used as a PreRunE function for all commands that
// require a project to be configured. If a project could successfully
// be found and loaded, it is set on the command's context object.
func Configure(cmd *cobra.Command, args []string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}

	ctx, err := Initialize(cmd.Context(), root, getEnvironment(cmd))
	if err != nil {
		return err
	}

	cmd.SetContext(ctx)
	return nil
}

// Placeholder to use as unique key in context.Context.
var projectKey int

// Initialize loads a project configuration given a root and environment.
// It stores the project on a new context.
// The project is available through the `Get()` function.
func Initialize(ctx context.Context, root, env string) (context.Context, error) {
	config, err := loadProjectConf(root)
	if err != nil {
		return nil, err
	}

	// Confirm that the specified environment is valid.
	environment, ok := config.Environments[env]
	if !ok {
		return nil, fmt.Errorf("environment [%s] not defined", env)
	}

	fileSet := git.NewFileSet(root)
	err = fileSet.EnsureValidGitIgnoreExists()
	if err != nil {
		return ctx, nil
	}

	p := project{
		root: root,
		env:  env,

		config:      &config,
		environment: &environment,
		fileSet:     fileSet,
	}

	p.initializeWorkspacesClient(ctx)
	return context.WithValue(ctx, &projectKey, &p), nil
}

func (p *project) initializeWorkspacesClient(ctx context.Context) {
	var config databricks.Config

	// If the config specifies a profile, or other authentication related properties,
	// pass them along to the SDK here. If nothing is defined, the SDK will figure
	// out which autentication mechanism to use using enviroment variables.
	if p.environment.Workspace.Profile != "" {
		config.Profile = p.environment.Workspace.Profile
	}

	p.wsc = workspaces.New(&config)
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

func (p *project) GetFileSet() *git.FileSet {
	return p.fileSet
}

// This cache dir will contain any state, state overrides (per user overrides
// to the project config) or any generated artifacts (eg: sync snapshots)
// that should never be checked into Git.
//
// We enfore that cache dir (.databricks) is added to .gitignore
// because it contains per-user overrides that we do not want users to
// accidentally check into git
func (p *project) CacheDir() (string, error) {
	// assert cache dir is present in git ignore
	if !p.fileSet.IsGitIgnored(fmt.Sprintf("/%s/", CacheDirName)) {
		return "", fmt.Errorf("please add /%s/ to .gitignore", CacheDirName)
	}

	cacheDirPath := filepath.Join(p.root, CacheDirName)
	// create cache dir if it does not exist
	if _, err := os.Stat(cacheDirPath); os.IsNotExist(err) {
		err = os.Mkdir(cacheDirPath, os.ModeDir|os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create cache directory %s with error: %s", cacheDirPath, err)
		}
	}
	return cacheDirPath, nil
}

func (p *project) Config() Config {
	return *p.config
}

func (p *project) Environment() Environment {
	return *p.environment
}

func (p *project) Me() (*scim.User, error) {
	// QQ: Why is there a lock here?
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

func runCommandOnDev(ctx context.Context, language, command string) commands.Results {
	clusterId, err := Get(ctx).GetDevelopmentClusterId(ctx)
	if err != nil {
		return commands.Results{
			ResultType: "error",
			Summary:    err.Error(),
		}
	}
	return Get(ctx).wsc.CommandExecutor.Execute(ctx, clusterId, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) commands.Results {
	return runCommandOnDev(ctx, "python", command)
}
