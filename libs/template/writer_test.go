package template

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWriterConfigure(t *testing.T) {
	// Test on local file system.
	w := &defaultWriter{}
	err := w.Configure(context.Background(), "/foo/bar", "/out/abc")
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

	ctx := dbr.MockRuntime(context.Background(), true)
	ctx = root.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	})
	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", "/Workspace/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w.configPath)
	assert.IsType(t, &filer.WorkspaceFilesExtensionsClient{}, w.outputFiler)
}

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	ctx := context.Background()

	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", tmpDir1)
	require.NoError(t, err)

	// Try to materialize a non-template directory.
	err = w.Materialize(ctx, &localReader{path: tmpDir2})
	assert.EqualError(t, err, "not a bundle template: expected to find a template schema file at databricks_template_schema.json")
}

func TestDefaultWriterLogTelemetry(t *testing.T) {
	ctx := telemetry.WithMockLogger(context.Background())
	w := &defaultWriter{templateName: Custom}
	w.LogTelemetry(ctx)

	logs := telemetry.Introspect(ctx)
	assert.Len(t, logs, 1)
	assert.Equal(t, &protos.BundleInitEvent{
		TemplateName: string(Custom),
		Uuid:         bundleUuid,
	}, logs[0].BundleInitEvent)
}

func TestWriterWithFullTelemetry(t *testing.T) {
	ctx := telemetry.WithMockLogger(context.Background())
	w := &writerWithFullTelemetry{
		defaultWriter: defaultWriter{
			templateName: DefaultPython,
			config: &config{
				values: map[string]any{
					"foo": "v1",
					"bar": "v2",
				},
				schema: &jsonschema.Schema{
					Properties: map[string]*jsonschema.Schema{
						"foo": {
							Type: jsonschema.StringType,
							Enum: []any{"v1", "v2"},
						},
						"bar": {
							Type: jsonschema.StringType,
						},
					},
				},
			},
		},
	}
	w.LogTelemetry(ctx)

	logs := telemetry.Introspect(ctx)
	assert.Len(t, logs, 1)
	assert.Equal(t, &protos.BundleInitEvent{
		TemplateName: string(DefaultPython),
		TemplateEnumArgs: []protos.BundleInitTemplateEnumArg{
			{
				Key:   "foo",
				Value: "v1",
			},
		},
		Uuid: bundleUuid,
	}, logs[0].BundleInitEvent)
}
