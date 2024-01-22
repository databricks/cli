package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func NewGeneratePipelineCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var pipelineId string
	var force bool

	cmd := &cobra.Command{
		Use:     "pipeline",
		Short:   "Generate bundle configuration for a pipeline",
		PreRunE: root.MustConfigureBundle,
	}

	cmd.Flags().StringVar(&pipelineId, "existing-pipeline-id", "", `ID of the pipeline to generate config for`)
	cmd.MarkFlagRequired("existing-pipeline-id")

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", filepath.Join(wd, "resources"), `Dir path where the output config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", filepath.Join(wd, "src"), `Dir path where the downloaded files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)
		w := b.WorkspaceClient()

		pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
		if err != nil {
			return err
		}

		downloader := newNotebookDownloader(w, sourceDir, configDir)
		for _, lib := range pipeline.Spec.Libraries {
			err := downloader.MarkLibraryForDownload(ctx, &lib)
			if err != nil {
				return err
			}
		}

		v, err := generate.ConvertPipelineToValue(pipeline.Spec)
		if err != nil {
			return err
		}

		jobKey := fmt.Sprintf("pipeline_%s", textutil.NormalizeString(pipeline.Name))
		result := map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"pipelines": dyn.V(map[string]dyn.Value{
					jobKey: v,
				}),
			}),
		}

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		filename := filepath.Join(configDir, fmt.Sprintf("%s.yml", jobKey))
		err = yamlsaver.SaveAsYAML(result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Pipeline configuration successfully saved to %s", filename))
		return nil
	}

	return cmd
}
