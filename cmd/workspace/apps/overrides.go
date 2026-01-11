package apps

import (
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

func deployOverride(deployCmd *cobra.Command, deployReq *apps.CreateAppDeploymentRequest) {
	originalRunE := deployCmd.RunE
	deployCmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := originalRunE(cmd, args)
		return wrapDeploymentError(cmd, deployReq.AppName, err)
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
		cmd.AddCommand(newLogsCommand())
	})
	listOverrides = append(listOverrides, listOverride)
	listDeploymentsOverrides = append(listDeploymentsOverrides, listDeploymentsOverride)
	createOverrides = append(createOverrides, createOverride)
	deployOverrides = append(deployOverrides, deployOverride)
	createUpdateOverrides = append(createUpdateOverrides, createUpdateOverride)
	startOverrides = append(startOverrides, startOverride)
}
