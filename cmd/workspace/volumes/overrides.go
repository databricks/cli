package volumes

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListVolumesRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .Name}}	{{.VolumeType}}	{{.FullName}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Name", Extract: func(v any) string {
			return v.(catalog.VolumeInfo).Name
		}},
		{Header: "Volume Type", Extract: func(v any) string {
			return string(v.(catalog.VolumeInfo).VolumeType)
		}},
		{Header: "Full Name", Extract: func(v any) string {
			return v.(catalog.VolumeInfo).FullName
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
