package pipelines

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndParsePath(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	pipelineDir := filepath.Join(srcDir, "my_pipeline")
	require.NoError(t, os.MkdirAll(pipelineDir, 0o755))

	sparkPipelineFile := filepath.Join(pipelineDir, "spark-pipeline.yml")
	require.NoError(t, os.WriteFile(sparkPipelineFile, []byte("name: test"), 0o644))

	t.Chdir(tmpDir)

	t.Run("ValidRelativePath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline")
		require.NoError(t, err)

		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)
		assert.Equal(t, "", info.sparkPipelineFile)
	})

	t.Run("ValidPathWithDotSlash", func(t *testing.T) {
		info, err := validateAndParsePath("./src/my_pipeline")
		require.NoError(t, err)

		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)
		assert.Equal(t, "", info.sparkPipelineFile)
	})

	t.Run("ValidAbsolutePath", func(t *testing.T) {
		absPath, err := filepath.Abs(filepath.Join("src", "my_pipeline"))
		require.NoError(t, err)

		info, err := validateAndParsePath(absPath)
		require.NoError(t, err)

		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)
		assert.Equal(t, "", info.sparkPipelineFile)
	})

	t.Run("ValidNonNormalizedPath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline/../my_pipeline")
		require.NoError(t, err)

		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)
		assert.Equal(t, "", info.sparkPipelineFile)
	})

	t.Run("ValidDirectFilePath", func(t *testing.T) {
		info, err := validateAndParsePath("src/my_pipeline/spark-pipeline.yml")
		require.NoError(t, err)

		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)

		// Resolve both paths to handle symlinks (e.g., /var -> /private/var on macOS)
		expectedAbs, err := filepath.EvalSymlinks(sparkPipelineFile)
		require.NoError(t, err)

		actualAbs, err := filepath.EvalSymlinks(info.sparkPipelineFile)
		require.NoError(t, err)

		assert.Equal(t, expectedAbs, actualAbs)
	})

	t.Run("ValidDirectFilePathWithSuffix", func(t *testing.T) {
		prodFile := filepath.Join(pipelineDir, "prod.spark-pipeline.yml")
		require.NoError(t, os.WriteFile(prodFile, []byte("name: test"), 0o644))
		defer os.Remove(prodFile)

		info, err := validateAndParsePath("src/my_pipeline/prod.spark-pipeline.yml")
		require.NoError(t, err)
		assert.Equal(t, "my_pipeline", info.directoryName)
		assert.Equal(t, filepath.Join("src", "my_pipeline"), info.pipelineDirectoryPath)

		// Resolve both paths to handle symlinks (e.g., /var -> /private/var on macOS)
		expectedAbs, err := filepath.EvalSymlinks(prodFile)
		require.NoError(t, err)
		actualAbs, err := filepath.EvalSymlinks(info.sparkPipelineFile)
		require.NoError(t, err)
		assert.Equal(t, expectedAbs, actualAbs)
	})

	t.Run("InvalidPathNotInSrc", func(t *testing.T) {
		_, err := validateAndParsePath("my_pipeline")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "folder must be directly in 'src' directory")
	})

	t.Run("InvalidPathJustSrc", func(t *testing.T) {
		_, err := validateAndParsePath("src")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "folder must be directly in 'src' directory")
	})

	t.Run("InvalidPathTooDeep", func(t *testing.T) {
		deepDir := filepath.Join(srcDir, "folder", "subfolder")
		require.NoError(t, os.MkdirAll(deepDir, 0o755))

		_, err := validateAndParsePath("src/folder/subfolder")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "folder must be directly in 'src' directory")
	})

	t.Run("InvalidPathOutsideProject", func(t *testing.T) {
		_, err := validateAndParsePath("../other_project/src/my_pipeline")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "folder must be within the current project directory")
	})
}

func TestFindSparkPipelineFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("SingleFile", func(t *testing.T) {
		pipelineDir := filepath.Join(tmpDir, "single")
		require.NoError(t, os.MkdirAll(pipelineDir, 0o755))

		sparkFile := filepath.Join(pipelineDir, "spark-pipeline.yml")
		require.NoError(t, os.WriteFile(sparkFile, []byte("name: test"), 0o644))

		found, err := findSparkPipelineFile(pipelineDir)
		require.NoError(t, err)
		assert.Equal(t, sparkFile, found)
	})

	t.Run("FileWithSuffix", func(t *testing.T) {
		pipelineDir := filepath.Join(tmpDir, "suffix")
		require.NoError(t, os.MkdirAll(pipelineDir, 0o755))

		prodFile := filepath.Join(pipelineDir, "prod.spark-pipeline.yml")
		require.NoError(t, os.WriteFile(prodFile, []byte("name: test"), 0o644))

		found, err := findSparkPipelineFile(pipelineDir)
		require.NoError(t, err)
		assert.Equal(t, prodFile, found)
	})

	t.Run("NoFile", func(t *testing.T) {
		pipelineDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.MkdirAll(pipelineDir, 0o755))

		_, err := findSparkPipelineFile(pipelineDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no spark-pipeline.yml")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		pipelineDir := filepath.Join(tmpDir, "multiple")
		require.NoError(t, os.MkdirAll(pipelineDir, 0o755))

		sparkFile := filepath.Join(pipelineDir, "spark-pipeline.yml")
		require.NoError(t, os.WriteFile(sparkFile, []byte("name: test"), 0o644))

		prodFile := filepath.Join(pipelineDir, "prod.spark-pipeline.yml")
		require.NoError(t, os.WriteFile(prodFile, []byte("name: test"), 0o644))

		_, err := findSparkPipelineFile(pipelineDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple spark-pipeline.yml files found")
	})
}
