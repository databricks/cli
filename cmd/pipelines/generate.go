package pipelines

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type sdpPipeline struct {
	name          string               `yaml:"name"`
	storage       string               `yaml:"storage,omitempty"`
	catalog       string               `yaml:"catalog,omitempty"`
	database      string               `yaml:"database,omitempty"`
	configuration map[string]string    `yaml:"configuration,omitempty"`
	libraries     []sdpPipelineLibrary `yaml:"libraries,omitempty"`
}

type sdpPipelineLibrary struct {
	glob sdpPipelineLibraryGlob `yaml:"glob,omitempty"`
}

type sdpPipelineLibraryGlob struct {
	include string `yaml:"include,omitempty"`
}

func generateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Generate Lakeflow SDP configuration from spark-pipeline.yml",
		Long: `Generate Lakeflow SDP configuration from spark-pipeline.yml.

The dir must be located directly in the 'src' directory (e.g., ./src/my_pipeline).
The command will find a spark-pipeline.yml or *.spark-pipeline.yml file in the folder
and generate a corresponding .pipeline.yml file in the resources directory. If multiple
files spark-pipeline.yml files exist, you can specify path to *.spark-pipeline.yml file.`,
		Args: cobra.ExactArgs(1),
	}

	var force bool
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing pipeline configuration file.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		folderPath := args[0]
		ctx := cmd.Context()

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

		spec, err := parseSparkPipelineYAML(sparkPipelineFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", sparkPipelineFile, err)
		}

		outputFile := filepath.Join("resources", info.directoryName+".pipeline.yml")
		resourceName := info.directoryName
		resourcesMap, err := convertToResources(spec, resourceName, info.pipelineDirectoryPath)
		if err != nil {
			return fmt.Errorf("failed to construct .pipeline.yml: %w", err)
		}

		saver := yamlsaver.NewSaver()
		err = saver.SaveAsYAML(resourcesMap, outputFile, force)
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
		sparkPipelineFile = folderPath
		folderPath = filepath.Dir(folderPath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", folderPath, err)
	}

	// Get relative path from CWD
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	// Normalize to forward slashes for consistent parsing (works on Windows too)
	relPath = filepath.ToSlash(relPath)

	// Check if path starts with ".." (outside of CWD)
	if strings.HasPrefix(relPath, "..") {
		return nil, fmt.Errorf("folder must be within the current project directory, got: %s", folderPath)
	}

	// Split path to check structure
	parts := strings.Split(relPath, "/")

	// Must be src/<folder_name>
	if len(parts) != 2 || parts[0] != "src" {
		return nil, fmt.Errorf("folder must be directly in 'src' directory (e.g., ./src/my_pipeline), got: %s", folderPath)
	}

	pipelineName := parts[1]

	// Return the relative path as pipelineDirectoryPath (keep it relative for later use)
	srcFolder := filepath.FromSlash(relPath)

	// Convert sparkPipelineFile to absolute if it was specified
	if sparkPipelineFile != "" {
		sparkPipelineFile, err = filepath.Abs(sparkPipelineFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", sparkPipelineFile, err)
		}
	}

	return &sdpPathInfo{
		directoryName:         pipelineName,
		pipelineDirectoryPath: srcFolder,
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

	return filepath.Join(folder, candidates[0]), nil
}

// parseSparkPipelineYAML parses a spark-pipeline.yml file.
func parseSparkPipelineYAML(filePath string) (*sdpPipeline, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var spec sdpPipeline
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// convertToResources converts a spark-pipeline.yml spec to DAB YAML format with "resources" property
func convertToResources(spec *sdpPipeline, resourceName string, srcFolder string) (map[string]dyn.Value, error) {
	// YAML paths are relative to directory containing YAML file, in this case:
	// DAB YAML is in "./resources/<directoryName>.pipeline.yml"
	// SDP YAML is in "./<pipelineDirectoryPath>/spark-pipeline.yml"
	relativePath := filepath.Join("..", srcFolder)

	var catalog = "${var.catalog}"
	if spec.catalog != "" {
		catalog = spec.catalog
	}

	var schema = "${var.schema}"
	if spec.database != "" {
		schema = spec.database
	}

	var libraries []pipelines.PipelineLibrary
	environment := pipelines.PipelinesEnvironment{
		Dependencies: []string{
			"--editable ${workspace.file_path}",
		},
	}

	for _, lib := range spec.libraries {
		if lib.glob.include != "" {
			relativeIncludePath := filepath.Join(relativePath, lib.glob.include)

			libraries = append(libraries, pipelines.PipelineLibrary{
				Glob: &pipelines.PathPattern{Include: relativeIncludePath},
			})
		}
	}

	environmentDyn, err := convert.FromTyped(environment, dyn.NilValue)
	if err != nil {
		return nil, fmt.Errorf("failed to convert environments into dyn.Value: %w", err)
	}

	librariesDyn, err := convert.FromTyped(libraries, dyn.NilValue)
	if err != nil {
		return nil, fmt.Errorf("failed to convert libraries into dyn.Value: %w", err)
	}

	pipelineDyn := dyn.V(
		map[string]dyn.Value{
			"name":        dyn.V(spec.name),
			"catalog":     dyn.V(catalog),
			"schema":      dyn.V(schema),
			"rootPath":    dyn.V(relativePath),
			"serverless":  dyn.V(true),
			"libraries":   librariesDyn,
			"environment": environmentDyn,
		},
	)

	resourcesMap := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"pipelines": dyn.V(map[string]dyn.Value{
				resourceName: pipelineDyn,
			}),
		}),
	}

	_, diag := convert.Normalize(&config.Resources{}, dyn.V(resourcesMap))
	if len(diag) > 0 {
		return nil, fmt.Errorf("generated output doesn't match schema: %s", diag)
	}

	return resourcesMap, nil
}
