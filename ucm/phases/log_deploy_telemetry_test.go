package phases_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/config/variable"
	"github.com/databricks/cli/ucm/phases"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureEvent runs LogDeployTelemetry against an in-process testserver and
// returns the BundleDeployEvent that ucm uploads. The logger's logs slice is
// unexported, so a round-trip through Upload is the only inspection path.
func captureEvent(t *testing.T, u *ucm.Ucm, errMsg string) *protos.BundleDeployEvent {
	t.Helper()

	server := testserver.New(t)
	t.Cleanup(server.Close)

	var captured []byte
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		captured = append([]byte(nil), req.Body...)
		return map[string]any{
			"errors":          []string{},
			"numProtoSuccess": 1,
		}
	})

	ctx := telemetry.WithNewLogger(t.Context())
	phases.LogDeployTelemetry(ctx, u, errMsg)
	ctx = cmdctx.SetConfigUsed(ctx, &sdkconfig.Config{Host: server.URL, Token: "token"})
	require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))

	var body struct {
		ProtoLogs []string `json:"protoLogs"`
	}
	require.NoError(t, json.Unmarshal(captured, &body))
	require.Len(t, body.ProtoLogs, 1)

	var fl protos.FrontendLog
	require.NoError(t, json.Unmarshal([]byte(body.ProtoLogs[0]), &fl))
	return fl.Entry.DatabricksCliLog.BundleDeployEvent
}

func TestLogDeployTelemetry_EmptyUcm(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte("ucm:\n  name: example\n"))
	require.Empty(t, diags)
	u := &ucm.Ucm{Config: *cfg}

	ev := captureEvent(t, u, "")
	require.NotNil(t, ev)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", ev.BundleUuid)
	assert.Empty(t, ev.ErrorMessage)
	assert.Equal(t, int64(0), ev.ResourceCount)
	assert.Equal(t, int64(0), ev.ResourceSchemaCount)
	assert.Equal(t, int64(0), ev.ResourceVolumeCount)
	require.NotNil(t, ev.Experimental)
	assert.Equal(t, protos.BundleModeUnspecified, ev.Experimental.BundleMode)
	assert.Equal(t, protos.BundleDeployArtifactPathTypeUnspecified, ev.Experimental.WorkspaceArtifactPathType)
}

func TestLogDeployTelemetry_CountsResources(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: example
resources:
  catalogs:
    a: { name: a }
    b: { name: b }
  schemas:
    s1: { name: s1, catalog_name: a }
    s2: { name: s2, catalog_name: a }
    s3: { name: s3, catalog_name: a }
  volumes:
    v1: { name: v1, catalog_name: a, schema_name: s1 }
`))
	require.Empty(t, diags)
	u := &ucm.Ucm{Config: *cfg}

	ev := captureEvent(t, u, "")
	require.NotNil(t, ev)
	// ResourceCount mirrors bundle's pattern walk: every resource under
	// resources.<kind>.<name> is counted regardless of kind.
	assert.Equal(t, int64(6), ev.ResourceCount)
	assert.Equal(t, int64(3), ev.ResourceSchemaCount)
	assert.Equal(t, int64(1), ev.ResourceVolumeCount)
}

func TestLogDeployTelemetry_ScrubsAndTruncatesErrorMessage(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}

	// Path scrubbing kicks in: the absolute path is replaced with a token.
	ev := captureEvent(t, u, "failed to read /home/user/secrets/keyfile")
	require.NotNil(t, ev)
	assert.NotContains(t, ev.ErrorMessage, "/home/user/secrets")
	assert.Contains(t, ev.ErrorMessage, "REDACTED")

	// Truncation kicks in at 500 chars (post-scrub).
	long := strings.Repeat("x", 1000)
	ev = captureEvent(t, u, long)
	require.NotNil(t, ev)
	assert.Len(t, ev.ErrorMessage, 500)
}

func TestLogDeployTelemetry_VariableCounts(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"plain":   {Value: "v"},
				"complex": {Value: map[string]any{"k": "v"}},
				"lookup":  {Lookup: &variable.Lookup{}},
			},
		},
	}

	ev := captureEvent(t, u, "")
	require.NotNil(t, ev)
	require.NotNil(t, ev.Experimental)
	assert.Equal(t, int64(3), ev.Experimental.VariableCount)
	assert.Equal(t, int64(1), ev.Experimental.ComplexVariableCount)
	assert.Equal(t, int64(1), ev.Experimental.LookupVariableCount)
}

func TestLogDeployTelemetry_CountsIncludeUndeployedResources(t *testing.T) {
	// Counts reflect the declared configuration regardless of whether the
	// resource carries a populated state ID — matches bundle's policy.
	u := &ucm.Ucm{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"a": {ID: "deployed-id"},
					"b": {ID: ""}, // not yet deployed
				},
				Volumes: map[string]*resources.Volume{
					"v": {ID: ""},
				},
			},
		},
	}

	ev := captureEvent(t, u, "")
	require.NotNil(t, ev)
	assert.Equal(t, int64(2), ev.ResourceSchemaCount)
	assert.Equal(t, int64(1), ev.ResourceVolumeCount)
}
