package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelativePathTranslationDefault(t *testing.T) {
	b, diags := initializeTarget(t, "./relative_path_translation", "default")
	require.NoError(t, diags.Error())

	t0 := b.Config.Resources.Jobs["job"].Tasks[0]
	assert.Equal(t, "/remote/src/file1.py", t0.SparkPythonTask.PythonFile)
	t1 := b.Config.Resources.Jobs["job"].Tasks[1]
	assert.Equal(t, "/remote/src/file1.py", t1.SparkPythonTask.PythonFile)
}

func TestRelativePathTranslationOverride(t *testing.T) {
	b, diags := initializeTarget(t, "./relative_path_translation", "override")
	require.NoError(t, diags.Error())

	t0 := b.Config.Resources.Jobs["job"].Tasks[0]
	assert.Equal(t, "/remote/src/file2.py", t0.SparkPythonTask.PythonFile)
	t1 := b.Config.Resources.Jobs["job"].Tasks[1]
	assert.Equal(t, "/remote/src/file2.py", t1.SparkPythonTask.PythonFile)
}
