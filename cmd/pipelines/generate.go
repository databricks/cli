package pipelines

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func generateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate pipeline configuration",
		Long: `Generate pipeline configuration

Use --existing-pipeline-dir to generate pipeline configuration from spark-pipeline.yml

	The directory must be located directly in the 'src' directory (e.g., ./src/my_pipeline).
	The command will find a spark-pipeline.yml or *.spark-pipeline.yml file in the folder
	and generate a corresponding .pipeline.yml file in the resources directory. If multiple
	spark-pipeline.yml files exist, you can specify the full path to a specific *.spark-pipeline.yml file.`,
	}

	var existingPipelineDir string
	var force bool
	cmd.Flags().StringVar(&existingPipelineDir, "existing-pipeline-dir", "", "Path to the existing pipeline directory in 'src' (e.g., src/my_pipeline).")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing pipeline configuration file.")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if existingPipelineDir == "" {
			err := cmd.Help()
			if err != nil {
				return err
			}

			return errors.New("required flag \"existing-pipeline-dir\" not set")
		}

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		folderPath := existingPipelineDir

		info, err := validateAndParsePath(folderPath)
		if err != nil {
			return err
		}

		sparkPipelineFile := info.sparkPipelineFile
		if sparkPipelineFile == "" {
			sparkPipelineFile, err = findSparkPipelineFile(info.pipelineDirectoryPath)
			if err != nil {
				return err
			}
		}

		spec, err := parseSparkPipelineYAML(ctx, sparkPipelineFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", sparkPipelineFile, err)
		}

		outputFile := filepath.ToSlash(filepath.Join("resources", info.directoryName+".pipeline.yml"))
		resourceName := info.directoryName
		resourcesMap, err := convertToResources(spec, resourceName, info.pipelineDirectoryPath)
		if err != nil {
			return fmt.Errorf("failed to construct .pipeline.yml: %w", err)
		}

		err = structyaml.Save(outputFile, resourcesMap, force, nil)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Generated pipeline configuration: %s\n", outputFile))
		return nil
	}

	return cmd
}

// sdpPathInfo contains structured information about spark-pipeline.yml in src directory
type sdpPathInfo struct {
	// directoryName is name of pipeline directory, e.g., "my_pipeline"
	directoryName string

	// pipelineDirectoryPath is directory containing SDP pipeline, e.g., "src/my_pipeline"
	pipelineDirectoryPath string

	// sparkPipelineFile is either "spark-pipeline.yml" or has ".spark-pipeline.yml" suffix
	sparkPipelineFile string
}

// validateAndParsePath validates the folder path and returns path information.
func validateAndParsePath(folderPath string) (*sdpPathInfo, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	var sparkPipelineFile string

	// Check if this is a direct path to a spark-pipeline.yml file
	if strings.HasSuffix(folderPath, ".spark-pipeline.yml") || strings.HasSuffix(folderPath, "spark-pipeline.yml") {
		sparkPipelineFile = filepath.ToSlash(folderPath)
		folderPath = filepath.Dir(folderPath)
	}

	absFolderPath, err := filepath.Abs(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", folderPath, err)
	}

	normalizedFolderPath, err := filepath.Rel(cwd, absFolderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	normalizedFolderPath = filepath.ToSlash(normalizedFolderPath)

	// Specified folder must be src/<folder_name>
	parts := strings.Split(normalizedFolderPath, "/")
	if len(parts) != 2 || parts[0] != "src" {
		return nil, fmt.Errorf("please make sure the directory is located in side 'src/' (for example 'src/my_pipeline'), got: %s", folderPath)
	}

	pipelineName := parts[1]

	return &sdpPathInfo{
		directoryName:         pipelineName,
		pipelineDirectoryPath: normalizedFolderPath,
		sparkPipelineFile:     sparkPipelineFile,
	}, nil
}

// findSparkPipelineFile finds a spark-pipeline.yml or *.spark-pipeline.yml file in the folder.
func findSparkPipelineFile(folder string) (string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", folder, err)
	}

	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "spark-pipeline.yml" || strings.HasSuffix(name, ".spark-pipeline.yml") {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no spark-pipeline.yml or *.spark-pipeline.yml file found in %s", folder)
	}

	if len(candidates) > 1 {
		return "", fmt.Errorf("multiple spark-pipeline.yml files found in %s: %v. Please specify the full path to disambiguate", folder, candidates)
	}

	return filepath.ToSlash(filepath.Join(folder, candidates[0])), nil
}

// sdpPipeline contains parsed SDP spark-pipeline.yml
type sdpPipeline struct {
	Name          string               `json:"name"`
	Catalog       string               `json:"catalog,omitempty"`
	Database      string               `json:"database,omitempty"`
	Libraries     []sdpPipelineLibrary `json:"libraries,omitempty"`
	Storage       string               `json:"storage,omitempty"`
	Configuration map[string]string    `json:"configuration,omitempty"`
}

// sdpPipelineLibrary contains 'library' field in spark-pipeline.yml
type sdpPipelineLibrary struct {
	Glob sdpPipelineLibraryGlob `json:"glob,omitempty"`
}

// sdpPipelineLibrary contains 'library.glob' field in spark-pipeline.yml
type sdpPipelineLibraryGlob struct {
	Include string `json:"include,omitempty"`
}

// parseSparkPipelineYAML parses a spark-pipeline.yml file.
func parseSparkPipelineYAML(ctx context.Context, filePath string) (*sdpPipeline, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer file.Close()

	out := sdpPipeline{}
	_, diags, err := structvar.DecodeYAML(filePath, file, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", filePath, err)
	}

	for _, diag := range diags {
		logdiag.LogDiag(ctx, diag)
	}

	return &out, nil
}

// convertToResources converts a spark-pipeline.yml spec to DABs YAML format with "resources" property
func convertToResources(spec *sdpPipeline, resourceName, srcFolder string) (structyaml.Map, error) {
	// YAML paths are relative to directory containing YAML file, in this case:
	// DABs YAML is in "./resources/<directoryName>.pipeline.yml"
	// SDP YAML is in "./<pipelineDirectoryPath>/spark-pipeline.yml"
	//
	// NB: all paths are /-based so Windows has the same output
	relativePath := filepath.ToSlash(filepath.Join("..", srcFolder))

	catalog := "${var.catalog}"
	if spec.Catalog != "" {
		catalog = spec.Catalog
	}

	schema := "${var.schema}"
	if spec.Database != "" {
		schema = spec.Database
	}

	environment := pipelines.PipelinesEnvironment{
		Dependencies: []string{
			"--editable ${workspace.file_path}",
		},
	}

	// Keys are written in the order they are added.
	pipelineMap := structyaml.M(
		"name", spec.Name,
		"catalog", catalog,
		"schema", schema,
		"root_path", relativePath,
		"serverless", true,
		"libraries", convertLibraries(relativePath, spec.Libraries),
	)

	// configuration is optional field, skip if empty
	if spec.Configuration != nil {
		pipelineMap.Add("configuration", structyaml.Value(spec.Configuration))
	}

	pipelineMap.Add("environment", structyaml.Value(environment))

	resourcesMap := structyaml.M("resources", structyaml.M("pipelines", structyaml.M(resourceName, pipelineMap)))

	_, diag, err := structvar.DecodeYAMLNode("", structyaml.Node(resourcesMap, nil), &config.Root{}, nil)
	if err != nil {
		return nil, err
	}
	if len(diag) > 0 {
		return nil, fmt.Errorf("generated output doesn't match expected schema: %v", diag)
	}

	return resourcesMap, nil
}

// convertLibraries converts SDP libraries into DABs YAML format
//
// relativePath contains a path to append into SDP libraries path to make
// them relative to generated DABs YAML
func convertLibraries(relativePath string, specLibraries []sdpPipelineLibrary) any {
	var libraries []pipelines.PipelineLibrary

	for _, lib := range specLibraries {
		if lib.Glob.Include != "" {
			relativeIncludePath := filepath.ToSlash(filepath.Join(relativePath, lib.Glob.Include))

			libraries = append(libraries, pipelines.PipelineLibrary{
				Glob: &pipelines.PathPattern{Include: relativeIncludePath},
			})
		}
	}

	// Value returns nil if libraries is an empty array
	if v := structyaml.Value(libraries); v != nil {
		return v
	}

	// we always want to leave empty array as a placeholder in generated YAML
	return []any{}
}
