package workspace

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *workspace.ListWorkspaceRequest) {
	listReq.Path = "/"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Type"}}	{{header "Language"}}	{{header "Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path|cyan}}
	{{end}}`)
}

func exportOverride(exportCmd *cobra.Command, exportReq *workspace.ExportRequest) {
	// The export command prints the contents of the file to stdout by default.
	exportCmd.Annotations["template"] = `{{.Content | b64_decode}}`
	exportCmd.Use = "export SOURCE_PATH"

	var filePath string
	exportCmd.Flags().StringVar(&filePath, "file", "", `Path on the local file system to save exported file at.`)

	exportCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No SOURCE_PATH argument specified. Loading names for Workspace drop-down."
			names, err := w.Workspace.ObjectInfoPathToObjectIdMap(ctx, workspace.ListWorkspaceRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Workspace drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The absolute path of the object or directory")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the absolute path of the object or directory")
		}
		exportReq.Path = args[0]

		response, err := w.Workspace.Export(ctx, *exportReq)
		if err != nil {
			return err
		}
		// Render file content to stdout if no file path is specified.
		if filePath == "" {
			return cmdio.Render(ctx, response)
		}
		b, err := base64.StdEncoding.DecodeString(response.Content)
		if err != nil {
			return err
		}
		return os.WriteFile(filePath, b, 0755)
	}
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	exportOverrides = append(exportOverrides, exportOverride)
}
