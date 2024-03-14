package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathTranslationFallback(t *testing.T) {
	b := loadTarget(t, "./path_translation/fallback", "development")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	require.NoError(t, err)

	j := b.Config.Resources.Jobs["my_job"]
	assert.Len(t, j.Tasks, 6)

	assert.Equal(t, "notebook_example", j.Tasks[0].TaskKey)
	assert.Equal(t, "src/notebook", j.Tasks[0].NotebookTask.NotebookPath)

	assert.Equal(t, "spark_python_example", j.Tasks[1].TaskKey)
	assert.Equal(t, "src/file.py", j.Tasks[1].SparkPythonTask.PythonFile)

	assert.Equal(t, "dbt_example", j.Tasks[2].TaskKey)
	assert.Equal(t, "src/dbt_project", j.Tasks[2].DbtTask.ProjectDirectory)

	assert.Equal(t, "sql_example", j.Tasks[3].TaskKey)
	assert.Equal(t, "src/sql.sql", j.Tasks[3].SqlTask.File.Path)

	assert.Equal(t, "python_wheel_example", j.Tasks[4].TaskKey)
	assert.Equal(t, "dist/wheel1.whl", j.Tasks[4].Libraries[0].Whl)
	assert.Equal(t, "dist/wheel2.whl", j.Tasks[4].Libraries[1].Whl)

	assert.Equal(t, "spark_jar_example", j.Tasks[5].TaskKey)
	assert.Equal(t, "target/jar1.jar", j.Tasks[5].Libraries[0].Jar)
	assert.Equal(t, "target/jar2.jar", j.Tasks[5].Libraries[1].Jar)

	p := b.Config.Resources.Pipelines["my_pipeline"]
	assert.Len(t, p.Libraries, 4)

	assert.Equal(t, "src/file1.py", p.Libraries[0].File.Path)
	assert.Equal(t, "src/notebook1", p.Libraries[1].Notebook.Path)
	assert.Equal(t, "src/file2.py", p.Libraries[2].File.Path)
	assert.Equal(t, "src/notebook2", p.Libraries[3].Notebook.Path)
}

func TestPathTranslationFallbackError(t *testing.T) {
	// TODO: add target with a bad path to trigger the error message
}

func TestPathTranslationNative(t *testing.T) {
	b := loadTarget(t, "./path_translation/native", "development")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, err)

	// TODO: assert output
}

func TestPathTranslationNativeError(t *testing.T) {
	// TODO: add target with a bad path to trigger the error message
}
