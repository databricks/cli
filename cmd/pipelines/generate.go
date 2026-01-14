package pipelines

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
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
		folderPath := existingPipelineDir
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

		outputFile := filepath.ToSlash(filepath.Join("resources", info.directoryName+".pipeline.yml"))
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
func parseSparkPipelineYAML(filePath string) (*sdpPipeline, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer file.Close()

	dv, err := yamlloader.LoadYAML(filePath, file)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", filePath, err)
	}

	out := sdpPipeline{}
	err = convert.ToTyped(&out, dv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	return &out, nil
}

// convertToResources converts a spark-pipeline.yml spec to DABs YAML format with "resources" property
func convertToResources(spec *sdpPipeline, resourceName, srcFolder string) (map[string]dyn.Value, error) {
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

	var libraries []pipelines.PipelineLibrary
	environment := pipelines.PipelinesEnvironment{
		Dependencies: []string{
			"--editable ${workspace.file_path}",
		},
	}

	for _, lib := range spec.Libraries {
		if lib.Glob.Include != "" {
			relativeIncludePath := filepath.ToSlash(filepath.Join(relativePath, lib.Glob.Include))

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

	// maps are unordered, and saver is sorting keys by dyn.Location
	// this is helper function to monotonically assign locations as keys are created
	var line int
	nextLocation := func() []dyn.Location {
		line += 1
		return []dyn.Location{{Line: line}}
	}

	pipelineMap := map[string]dyn.Value{
		"name":       dyn.V(spec.Name).WithLocations(nextLocation()),
		"catalog":    dyn.V(catalog).WithLocations(nextLocation()),
		"schema":     dyn.V(schema).WithLocations(nextLocation()),
		"root_path":  dyn.V(relativePath).WithLocations(nextLocation()),
		"serverless": dyn.V(true).WithLocations(nextLocation()),
		"libraries":  librariesDyn.WithLocations(nextLocation()),
	}

	// configuration is optional field, skip if empty
	if spec.Configuration != nil {
		dv, err := convert.FromTyped(spec.Configuration, dyn.NilValue)
		if err != nil {
			return nil, fmt.Errorf("failed to convert configuration into dyn.Value: %w", err)
		}

		// NB: golang maps are unordered, and currently we don't preserve the order
		pipelineMap["configuration"] = dv.WithLocations(nextLocation())
	}

	pipelineMap["environment"] = environmentDyn.WithLocations(nextLocation())

	resourcesMap := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"pipelines": dyn.V(map[string]dyn.Value{
				resourceName: dyn.V(pipelineMap),
			}),
		}),
	}

	_, diag := convert.Normalize(&config.Root{}, dyn.V(resourcesMap))
	if len(diag) > 0 {
		return nil, fmt.Errorf("generated output doesn't match expected schema: %v", diag)
	}

	return resourcesMap, nil
}
