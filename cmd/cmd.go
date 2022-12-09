package cmd

import (
	"github.com/databricks/bricks/cmd/root"

	"github.com/databricks/bricks/cmd/billing"
	"github.com/databricks/bricks/cmd/clusterpolicies"
	"github.com/databricks/bricks/cmd/clusters"
	"github.com/databricks/bricks/cmd/commands"
	"github.com/databricks/bricks/cmd/dbfs"
	"github.com/databricks/bricks/cmd/deployment"
	"github.com/databricks/bricks/cmd/gitcredentials"
	"github.com/databricks/bricks/cmd/globalinitscripts"
	"github.com/databricks/bricks/cmd/instancepools"
	"github.com/databricks/bricks/cmd/ipaccesslists"
	"github.com/databricks/bricks/cmd/jobs"
	"github.com/databricks/bricks/cmd/libraries"
	"github.com/databricks/bricks/cmd/mlflow"
	"github.com/databricks/bricks/cmd/permissions"
	"github.com/databricks/bricks/cmd/pipelines"
	"github.com/databricks/bricks/cmd/repos"
	"github.com/databricks/bricks/cmd/scim"
	"github.com/databricks/bricks/cmd/secrets"
	"github.com/databricks/bricks/cmd/sql"
	"github.com/databricks/bricks/cmd/tokenmanagement"
	"github.com/databricks/bricks/cmd/tokens"
	"github.com/databricks/bricks/cmd/unitycatalog"
	"github.com/databricks/bricks/cmd/workspace"
	"github.com/databricks/bricks/cmd/workspaceconf"
)

func init() {

	root.RootCmd.AddCommand(billing.Cmd)
	root.RootCmd.AddCommand(clusterpolicies.Cmd)
	root.RootCmd.AddCommand(clusters.Cmd)
	root.RootCmd.AddCommand(commands.Cmd)
	root.RootCmd.AddCommand(dbfs.Cmd)
	root.RootCmd.AddCommand(deployment.Cmd)
	root.RootCmd.AddCommand(gitcredentials.Cmd)
	root.RootCmd.AddCommand(globalinitscripts.Cmd)
	root.RootCmd.AddCommand(instancepools.Cmd)
	root.RootCmd.AddCommand(ipaccesslists.Cmd)
	root.RootCmd.AddCommand(jobs.Cmd)
	root.RootCmd.AddCommand(libraries.Cmd)
	root.RootCmd.AddCommand(mlflow.Cmd)
	root.RootCmd.AddCommand(permissions.Cmd)
	root.RootCmd.AddCommand(pipelines.Cmd)
	root.RootCmd.AddCommand(repos.Cmd)
	root.RootCmd.AddCommand(scim.Cmd)
	root.RootCmd.AddCommand(secrets.Cmd)
	root.RootCmd.AddCommand(sql.Cmd)
	root.RootCmd.AddCommand(tokenmanagement.Cmd)
	root.RootCmd.AddCommand(tokens.Cmd)
	root.RootCmd.AddCommand(unitycatalog.Cmd)
	root.RootCmd.AddCommand(workspace.Cmd)
	root.RootCmd.AddCommand(workspaceconf.Cmd)
}
