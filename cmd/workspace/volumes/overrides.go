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
		tableview.Col("Name", func(vol catalog.VolumeInfo) string { return vol.Name }),
		tableview.Col("Volume Type", func(vol catalog.VolumeInfo) string { return string(vol.VolumeType) }),
		tableview.Col("Full Name", func(vol catalog.VolumeInfo) string { return vol.FullName }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
