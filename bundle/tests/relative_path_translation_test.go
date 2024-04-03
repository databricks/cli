package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func configureMock(t *testing.T, b *bundle.Bundle) {
	// Configure mock workspace client
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{
		Host: "https://mock.databricks.workspace.com",
	}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "user@domain.com",
	}, nil)
	b.SetWorkpaceClient(m.WorkspaceClient)
}

func TestRelativePathTranslationDefault(t *testing.T) {
	b := loadTarget(t, "./relative_path_translation", "default")
	configureMock(t, b)

	diags := bundle.Apply(context.Background(), b, phases.Initialize())
	require.NoError(t, diags.Error())

	t0 := b.Config.Resources.Jobs["job"].Tasks[0]
	assert.Equal(t, "/remote/src/file1.py", t0.SparkPythonTask.PythonFile)
	t1 := b.Config.Resources.Jobs["job"].Tasks[1]
	assert.Equal(t, "/remote/src/file1.py", t1.SparkPythonTask.PythonFile)
}

func TestRelativePathTranslationOverride(t *testing.T) {
	b := loadTarget(t, "./relative_path_translation", "override")
	configureMock(t, b)

	diags := bundle.Apply(context.Background(), b, phases.Initialize())
	require.NoError(t, diags.Error())

	t0 := b.Config.Resources.Jobs["job"].Tasks[0]
	assert.Equal(t, "/remote/src/file2.py", t0.SparkPythonTask.PythonFile)
	t1 := b.Config.Resources.Jobs["job"].Tasks[1]
	assert.Equal(t, "/remote/src/file2.py", t1.SparkPythonTask.PythonFile)
}
