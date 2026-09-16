package template

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWriterConfigure(t *testing.T) {
	// Test on local file system.
	w := &defaultWriter{}
	err := w.Configure(t.Context(), "/foo/bar", "/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w.configPath)
	assert.IsType(t, &filer.LocalClient{}, w.outputFiler)
}

func TestDefaultWriterConfigureOnDBR(t *testing.T) {
	// This test is not valid on windows because a DBR image is always based on
	// Linux.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	ctx := dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: true, Version: "15.4"})
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.test"},
	})
	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", "/Workspace/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w.configPath)
	assert.IsType(t, &filer.WorkspaceFilesExtensionsClient{}, w.outputFiler)
}

func TestWriterWithFullTelemetryTemplateEnumArgs(t *testing.T) {
	// Enum values can be strings, integers or numbers; booleans are always
	// reported. Non-enum, non-boolean values are not reported.
	w := &writerWithFullTelemetry{}
	w.config = &config{
		values: map[string]any{
			"str_enum": "v2",
			"int_enum": int64(2),
			"num_enum": 1.5,
			"flag":     true,
			"freeform": "some-user-value",
		},
		schema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"str_enum": {Type: jsonschema.StringType, Enum: []any{"v1", "v2"}},
				"int_enum": {Type: jsonschema.IntegerType, Enum: []any{int64(1), int64(2)}},
				"num_enum": {Type: jsonschema.NumberType, Enum: []any{1.5, 2.5}},
				"flag":     {Type: jsonschema.BooleanType},
				"freeform": {Type: jsonschema.StringType},
			},
		},
	}

	assert.Equal(t, []protos.BundleInitTemplateEnumArg{
		{Key: "flag", Value: "true"},
		{Key: "int_enum", Value: "2"},
		{Key: "num_enum", Value: "1.5"},
		{Key: "str_enum", Value: "v2"},
	}, w.templateEnumArgs())
}

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	ctx := t.Context()

	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", tmpDir1)
	require.NoError(t, err)

	// Try to materialize a non-template directory.
	err = w.Materialize(ctx, &localReader{path: tmpDir2})
	assert.EqualError(t, err, "not a bundle template: expected to find a template schema file at databricks_template_schema.json")
}
