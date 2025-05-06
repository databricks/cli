package telemetry

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryUploadRetriesOnPartialSuccess(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return ResponseBody{
				NumProtoSuccess: 1,
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	ctx := WithNewLogger(context.Background())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})

	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})

	err := Upload(ctx, protos.ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func uploadRetriesFor(t *testing.T, statusCode int) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return testserver.Response{
				StatusCode: statusCode,
				Body: apierr.APIError{
					StatusCode: statusCode,
					Message:    "Some error",
				},
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

	ctx := WithNewLogger(context.Background())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})

	err := Upload(ctx, protos.ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestTelemetryUploadRetriesForStatusCodes(t *testing.T) {
	// These retries happen in the CLI itself since the SDK does not automatically
	// retry for 5xx errors.
	uploadRetriesFor(t, 500)
	uploadRetriesFor(t, 504)

	// These retries happen on the SDK layer.
	// ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
	uploadRetriesFor(t, 503)
	uploadRetriesFor(t, 429)
}

func TestTelemetryUploadMaxRetries(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	count := 0

	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		return ResponseBody{
			NumProtoSuccess: 1,
		}
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")
	ctx := WithNewLogger(context.Background())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})

	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})

	err := Upload(ctx, protos.ExecutionContext{})
	assert.EqualError(t, err, "failed to upload telemetry logs after three attempts")
	assert.Equal(t, 3, count)
}
