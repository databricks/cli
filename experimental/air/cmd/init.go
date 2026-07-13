package aircmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/experimental/air/trainyaml"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	var fromPath string
	var outputDir string
	var jobKey string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a Databricks Asset Bundle from an AIR CLI train.yaml",
		Long: `Scaffold a Databricks Asset Bundle from an AIR CLI train.yaml.

This reads an existing AIR CLI train.yaml and generates a bundle that deploys the
same workload as a durable job using an AI Runtime task. The generated bundle
contains:

  - databricks.yml         the bundle definition and a dev target
  - resources/<key>.job.yml the job with an ai_runtime_task
  - <code_source>/command.sh the command materialized as a script

Deploy the generated bundle with:
  databricks bundle deploy`,
	}

	cmd.Flags().StringVarP(&fromPath, "from", "f", "", "Path to the AIR CLI train.yaml to migrate")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated bundle into")
	cmd.Flags().StringVar(&jobKey, "key", "", "Resource key for the generated job (defaults to the experiment name)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files in the output directory")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := trainyaml.Parse(fromPath)
		if err != nil {
			return err
		}

		res, err := trainyaml.Convert(cfg)
		if err != nil {
			return err
		}

		for _, w := range res.Warnings {
			cmdio.LogString(ctx, "Warning: "+w)
		}

		if jobKey == "" {
			jobKey = textutil.NormalizeString(res.Job.Name)
		}

		if err := writeBundle(outputDir, jobKey, res, force); err != nil {
			return err
		}

		cmdio.LogString(ctx, "Bundle written to "+filepath.ToSlash(outputDir))
		cmdio.LogString(ctx, "Deploy it with: databricks bundle deploy")
		return nil
	}

	return cmd
}

// writeBundle materializes the generated bundle files under outputDir.
func writeBundle(outputDir, jobKey string, res *trainyaml.Result, force bool) error {
	jobValue, err := generate.ConvertJobToValue(&jobs.Job{Settings: &res.Job.JobSettings})
	if err != nil {
		return err
	}

	saver := yamlsaver.NewSaver()

	// yamlsaver orders map keys by their location line, so assign increasing
	// lines to fix the top-level key order (bundle, include, workspace).
	// Without this the order is nondeterministic (all keys default to line 0).
	rootValue := map[string]dyn.Value{
		"bundle": dyn.NewValue(map[string]dyn.Value{
			"name": dyn.V(jobKey),
		}, []dyn.Location{{Line: 0}}),
		"include": dyn.NewValue([]dyn.Value{dyn.V("resources/*.yml")}, []dyn.Location{{Line: 1}}),
	}
	// The snapshot's remote volume becomes the artifact path, so the code archive
	// is uploaded to the volume the user already configured for AIR.
	if res.ArtifactPath != "" {
		rootValue["workspace"] = dyn.NewValue(map[string]dyn.Value{
			"artifact_path": dyn.V(res.ArtifactPath),
		}, []dyn.Location{{Line: 2}})
	}

	resourcesDir := filepath.Join(outputDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		return err
	}

	if err := saver.SaveAsYAML(rootValue, filepath.Join(outputDir, "databricks.yml"), force); err != nil {
		return err
	}

	jobFile := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				jobKey: jobValue,
			}),
		}),
	}
	if err := saver.SaveAsYAML(jobFile, filepath.Join(resourcesDir, jobKey+".job.yml"), force); err != nil {
		return err
	}

	// Materialize the command next to the code it runs against, so command_path
	// (which is relative to the extracted code source) resolves at runtime.
	codeDir := filepath.Join(outputDir, filepath.FromSlash(codeSourceRootPath(res)))
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	commandPath := filepath.Join(codeDir, "command.sh")
	if !force {
		if _, err := os.Stat(commandPath); err == nil {
			return fmt.Errorf("%s already exists, use --force to overwrite", filepath.ToSlash(commandPath))
		}
	}
	return os.WriteFile(commandPath, []byte(res.CommandScript), 0o755)
}

// codeSourceRootPath returns the local code_source_path of the generated job's
// AI Runtime task, i.e. the directory the command script is written into.
func codeSourceRootPath(res *trainyaml.Result) string {
	return res.Job.Tasks[0].AiRuntimeTask.CodeSourcePath
}
