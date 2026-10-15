package external_locations

import (
	"github.com/databricks/cli/libs/cmdio"
	ucexternallocations "github.com/databricks/cli/libs/sdktypes/uc/externallocations/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *ucexternallocations.ListExternalLocationsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Credential"}}	{{header "URL"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.CredentialName|cyan}}	{{.Url}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
