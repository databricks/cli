package debug

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

func NewRenderTemplateSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "render-template-schema [TEMPLATE_PATH]",
		Short:  "Render a template schema with provided input values",
		Args:   root.MaximumNArgs(1),
		Hidden: true,
	}

	var inputFile string
	var templateDir string
	var tag string
	var branch string

	cmd.Flags().StringVar(&inputFile, "input-file", "", "JSON file containing key value pairs of input parameters required for template schema rendering.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag to use for template initialization")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to use for template initialization")

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if len(args) == 0 {
			return errors.New("template path is required")
		}
		templatePathOrUrl := args[0]

		// Resolve git ref from tag/branch flags
		ref := branch
		if tag != "" {
			ref = tag
		}

		// Resolve the template reader
		reader, isGitReader := template.ResolveReader(templatePathOrUrl, templateDir, ref)
		defer reader.Cleanup(ctx)

		// For git reader, load schema first to initialize the temp directory
		if isGitReader {
			_, _, err := reader.LoadSchemaAndTemplateFS(ctx)
			if err != nil {
				return err
			}
		}

		// Render the schema
		result, err := template.RenderSchema(ctx, reader, template.RenderSchemaInput{
			InputFile: inputFile,
		})
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), result.Content)
		return err
	}

	return cmd
}
