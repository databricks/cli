package pipelines

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndParsePath(t *testing.T) {
	t.Run("ValidRelativePath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline")
		require.NoError(t, err)

		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "",
		}, info)
	})

	t.Run("ValidPathWithDotSlash", func(t *testing.T) {
		info, err := validateAndParsePath("./src/my_pipeline")
		require.NoError(t, err)

		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "",
		}, info)
	})

	t.Run("ValidAbsolutePath", func(t *testing.T) {
		absPath, err := filepath.Abs(filepath.Join("src", "my_pipeline"))
		require.NoError(t, err)

		info, err := validateAndParsePath(absPath)
		require.NoError(t, err)

		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "",
		}, info)
	})

	t.Run("ValidNonNormalizedPath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline/../my_pipeline")
		require.NoError(t, err)
		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "",
		}, info)
	})

	t.Run("ValidDirectFilePath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline/spark-pipeline.yml")
		require.NoError(t, err)

		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "src/my_pipeline/spark-pipeline.yml",
		}, info)
	})

	t.Run("ValidDirectFilePathWithSuffix", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline/prod.spark-pipeline.yml")
		require.NoError(t, err)

		assert.Equal(t, &sdpPathInfo{
			directoryName:         "my_pipeline",
			pipelineDirectoryPath: filepath.Join("src", "my_pipeline"),
			sparkPipelineFile:     "src/my_pipeline/prod.spark-pipeline.yml",
		}, info)
	})

	t.Run("InvalidPathNotInSrc", func(t *testing.T) {
		_, err := validateAndParsePath("my_pipeline")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pipeline folder must be moved into 'src' directory")
	})

	t.Run("InvalidPathJustSrc", func(t *testing.T) {
		_, err := validateAndParsePath("src")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pipeline folder must be moved into 'src' directory")
	})

	t.Run("InvalidPathTooDeep", func(t *testing.T) {
		_, err := validateAndParsePath("src/folder/subfolder")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pipeline folder must be moved into 'src' directory")
	})

	t.Run("InvalidPathOutsideProject", func(t *testing.T) {
		_, err := validateAndParsePath("../other_project/src/my_pipeline")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pipeline folder must be moved into 'src' directory")
	})
}

func TestFindSparkPipelineFile(t *testing.T) {
	t.Chdir(t.TempDir())

	t.Run("SingleFile", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("single", 0o755))
		require.NoError(t, os.WriteFile("single/spark-pipeline.yml", []byte("name: test"), 0o644))

		found, err := findSparkPipelineFile("single")
		require.NoError(t, err)

		assert.Equal(t, "single/spark-pipeline.yml", found)
	})

	t.Run("FileWithSuffix", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("suffix", 0o755))
		require.NoError(t, os.WriteFile("suffix/prod.spark-pipeline.yml", []byte("name: test"), 0o644))

		found, err := findSparkPipelineFile("suffix")
		require.NoError(t, err)

		assert.Equal(t, "suffix/prod.spark-pipeline.yml", found)
	})

	t.Run("NoFile", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("empty", 0o755))

		_, err := findSparkPipelineFile("empty")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "no spark-pipeline.yml")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("multiple", 0o755))

		require.NoError(t, os.WriteFile("multiple/spark-pipeline.yml", []byte("name: test"), 0o644))
		require.NoError(t, os.WriteFile("multiple/prod.spark-pipeline.yml", []byte("name: test"), 0o644))

		_, err := findSparkPipelineFile("multiple")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "multiple spark-pipeline.yml files found")
	})
}

func TestConvertToResources(t *testing.T) {
	input := sdpPipeline{
		name: "My Pipeline",
		configuration: map[string]string{
			"key0": "value0",
			"key1": "value1",
		},
		libraries: []sdpPipelineLibrary{
			{
				glob: sdpPipelineLibraryGlob{
					include: "transformations/**",
				},
			},
		},
	}

	expected := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"pipelines": dyn.V(map[string]dyn.Value{
				"my_pipeline": dyn.V(map[string]dyn.Value{
					"name":       dyn.V("My Pipeline").WithLocations([]dyn.Location{{Line: 1}}),
					"catalog":    dyn.V("${var.catalog}").WithLocations([]dyn.Location{{Line: 2}}),
					"schema":     dyn.V("${var.schema}").WithLocations([]dyn.Location{{Line: 3}}),
					"root_path":  dyn.V("../src/my_pipeline").WithLocations([]dyn.Location{{Line: 4}}),
					"serverless": dyn.V(true).WithLocations([]dyn.Location{{Line: 5}}),
					"libraries": dyn.V([]dyn.Value{
						dyn.V(map[string]dyn.Value{
							"glob": dyn.V(map[string]dyn.Value{
								"include": dyn.V("../src/my_pipeline/transformations/**"),
							}),
						}),
					}).WithLocations([]dyn.Location{{Line: 6}}),
					"configuration": dyn.V(map[string]dyn.Value{
						"key0": dyn.V("value0"),
						"key1": dyn.V("value1"),
					}).WithLocations([]dyn.Location{{Line: 7}}),
					"environment": dyn.V(map[string]dyn.Value{
						"dependencies": dyn.V([]dyn.Value{
							dyn.V("--editable ${workspace.file_path}"),
						}),
					}).WithLocations([]dyn.Location{{Line: 8}}),
				}),
			}),
		}),
	}

	actual, err := convertToResources(&input, "my_pipeline", "src/my_pipeline")
	require.NoError(t, err)

	dynassert.Equal(t, dyn.V(expected), dyn.V(actual))
}
