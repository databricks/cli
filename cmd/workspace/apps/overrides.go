package apps

import (
	"slices"

	appsCli "github.com/databricks/cli/cmd/apps"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *apps.ListAppsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Url"}}	{{header "ComputeStatus"}}	{{header "DeploymentStatus"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name | green}}	{{.Url}}	{{if .ComputeStatus}}{{if eq .ComputeStatus.State "ACTIVE"}}{{green "%s" .ComputeStatus.State }}{{else}}{{blue "%s" .ComputeStatus.State}}{{end}}{{end}}	{{if .ActiveDeployment}}{{if eq .ActiveDeployment.Status.State "SUCCEEDED"}}{{green "%s" .ActiveDeployment.Status.State }}{{else}}{{blue "%s" .ActiveDeployment.Status.State}}{{end}}{{end}}
	{{end}}`)
}

func listDeploymentsOverride(listDeploymentsCmd *cobra.Command, listDeploymentsReq *apps.ListAppDeploymentsRequest) {
	listDeploymentsCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "DeploymentId"}}	{{header "State"}}	{{header "CreatedAt"}}`)
	listDeploymentsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.DeploymentId}}	{{if eq .Status.State "SUCCEEDED"}}{{green "%s" .Status.State }}{{else}}{{blue "%s" .Status.State}}{{end}}	{{.CreateTime}}
	{{end}}`)
}

func createOverride(createCmd *cobra.Command, createReq *apps.CreateAppRequest) {
	createCmd.Short = `Create an app in your workspace.`
	createCmd.Long = `Create an app in your workspace.`

	originalRunE := createCmd.RunE
	createCmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := originalRunE(cmd, args)
		return wrapDeploymentError(cmd, createReq.App.Name, err)
	}
}

func createUpdateOverride(createUpdateCmd *cobra.Command, createUpdateReq *apps.AsyncUpdateAppRequest) {
	originalRunE := createUpdateCmd.RunE
	createUpdateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := originalRunE(cmd, args)
		return wrapDeploymentError(cmd, createUpdateReq.AppName, err)
	}
}

func startOverride(startCmd *cobra.Command, startReq *apps.StartAppRequest) {
	originalRunE := startCmd.RunE
	startCmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := originalRunE(cmd, args)
		return wrapDeploymentError(cmd, startReq.Name, err)
	}
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		// Commands that should NOT go into the management group
		// (either they are main commands or have special grouping)
		nonManagementCommands := []string{
			// 'deploy' is overloaded as API and bundle command
			"deploy",
			// permission commands are assigned into "permission" group in cmd/cmd.go
			"get-permission-levels",
			"get-permissions",
			"set-permissions",
			"update-permissions",
		}

		// Put auto-generated API commands into 'management' group
		for _, subCmd := range cmd.Commands() {
			if slices.Contains(nonManagementCommands, subCmd.Name()) {
				continue
			}
			if subCmd.GroupID == "" {
				subCmd.GroupID = appsCli.ManagementGroupID
			}
		}

		// Add custom commands from cmd/apps/
		for _, appsCmd := range appsCli.Commands() {
			cmd.AddCommand(appsCmd)
		}

		// Add --var flag support for bundle operations
		cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in bundle config. Example: --var="key=value"`)
	})

	// Register command overrides
	listOverrides = append(listOverrides, listOverride)
	listDeploymentsOverrides = append(listDeploymentsOverrides, listDeploymentsOverride)
	createOverrides = append(createOverrides, createOverride)
	deployOverrides = append(deployOverrides, appsCli.BundleDeployOverrideWithWrapper(wrapDeploymentError))
	createUpdateOverrides = append(createUpdateOverrides, createUpdateOverride)
	startOverrides = append(startOverrides, startOverride)
}
