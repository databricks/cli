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

func init() {
	listOverrides = append(listOverrides, listOverride)
	listDeploymentsOverrides = append(listDeploymentsOverrides, listDeploymentsOverride)
}
