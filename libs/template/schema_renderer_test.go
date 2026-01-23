package template

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSchemaWithLocalTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	schemaContent := `{
    "welcome_message": "Hello {{.name}}!",
    "properties": {
        "project_name": {
            "type": "string",
            "description": "Project name for {{.name}}"
        }
    }
}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	inputFile := filepath.Join(tmpDir, "input.json")
	testutil.WriteFile(t, inputFile, `{"name": "TestUser"}`)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{})

	reader := NewLocalReader(tmpDir)
	result, err := RenderSchema(ctx, reader, RenderSchemaInput{
		InputFile: inputFile,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, `"welcome_message": "Hello TestUser!"`)
	assert.Contains(t, result.Content, `"description": "Project name for TestUser"`)
}

func TestRenderSchemaWithoutInputFile(t *testing.T) {
	tmpDir := t.TempDir()
	schemaContent := `{
    "welcome_message": "Hello!",
    "properties": {
        "project_name": {
            "type": "string",
            "description": "Project name"
        }
    }
}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{})

	reader := NewLocalReader(tmpDir)
	result, err := RenderSchema(ctx, reader, RenderSchemaInput{})
	require.NoError(t, err)

	assert.Contains(t, result.Content, `"welcome_message": "Hello!"`)
}

func TestRenderSchemaWithMissingVariable(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `{
    "welcome_message": "Hello {{.missing_var}}!"
}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	inputFile := filepath.Join(tmpDir, "input.json")
	testutil.WriteFile(t, inputFile, `{}`)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{})

	reader := NewLocalReader(tmpDir)
	// Missing variables are rendered as empty strings
	result, err := RenderSchema(ctx, reader, RenderSchemaInput{
		InputFile: inputFile,
	})
	require.NoError(t, err)
	assert.Contains(t, result.Content, `"welcome_message": "Hello !"`)
}

func TestRenderSchemaWithInvalidInputFile(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `{"welcome_message": "Hello"}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{})

	reader := NewLocalReader(tmpDir)
	_, err := RenderSchema(ctx, reader, RenderSchemaInput{
		InputFile: "/nonexistent/input.json",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read input file")
}

func TestRenderSchemaWithMalformedInputJson(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `{"welcome_message": "Hello"}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	inputFile := filepath.Join(tmpDir, "input.json")
	testutil.WriteFile(t, inputFile, `{invalid json}`)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{})

	reader := NewLocalReader(tmpDir)
	_, err := RenderSchema(ctx, reader, RenderSchemaInput{
		InputFile: inputFile,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse input file")
}

func TestRenderSchemaWithBuiltinTemplateFS(t *testing.T) {
	ctx := context.Background()
	reader := NewBuiltinReader(string(DefaultPython))
	schemaFS, err := reader.SchemaFS(ctx)
	require.NoError(t, err)
	assert.NotNil(t, schemaFS)
}

func TestRenderSchemaWithSimpleBuiltinTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `{
    "welcome_message": "Welcome to {{workspace_host}}!",
    "properties": {
        "project_name": {
            "type": "string",
            "description": "Project name"
        }
    }
}`
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), schemaContent)

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{
			Host: "https://test.databricks.com",
		},
	})

	reader := NewLocalReader(tmpDir)
	result, err := RenderSchema(ctx, reader, RenderSchemaInput{})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "https://test.databricks.com")
	assert.Contains(t, result.Content, "welcome_message")
}

func TestLocalReaderSchemaFS(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, "test.txt"), "content")

	reader := NewLocalReader(tmpDir)
	ctx := context.Background()

	schemaFS, err := reader.SchemaFS(ctx)
	require.NoError(t, err)
	assert.NotNil(t, schemaFS)
}

func TestBuiltinReaderSchemaFS(t *testing.T) {
	reader := NewBuiltinReader(string(DefaultPython))
	ctx := context.Background()

	fs, err := reader.SchemaFS(ctx)
	require.NoError(t, err)
	assert.NotNil(t, fs)
}

func TestBuiltinReaderSchemaFSNotFound(t *testing.T) {
	reader := NewBuiltinReader("nonexistent-template")
	ctx := context.Background()

	_, err := reader.SchemaFS(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
