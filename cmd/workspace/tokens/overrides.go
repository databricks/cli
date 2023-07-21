package tokens

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Expiry time"}}	{{header "Comment"}}
	{{range .}}{{.TokenId|green}}	{{cyan "%d" .ExpiryTime}}	{{.Comment|cyan}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
