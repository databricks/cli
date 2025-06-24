package pipelines

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	var outputDir string
	var configFile string
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new pipelines project",
		PreRunE: root.MustWorkspaceClient,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			r := template.Resolver{
				TemplatePathOrUrl: "pipelines",
				ConfigFile:        configFile,
				OutputDir:         outputDir,
			}

			tmpl, err := r.Resolve(ctx)
			if err != nil {
				return err
			}
			defer tmpl.Reader.Cleanup(ctx)

			err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
			if err != nil {
				return err
			}
			tmpl.Writer.LogTelemetry(ctx)
			return nil
		},
	}
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")
	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON file containing key value pairs of input parameters required for template initialization")
	return cmd
}
