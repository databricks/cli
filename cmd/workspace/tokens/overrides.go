package tokens

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Expiry time"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.TokenId|green}}	{{cyan "%d" .ExpiryTime}}	{{.Comment|cyan}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
