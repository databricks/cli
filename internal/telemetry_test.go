package internal

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wrapper to capture the response from the API client since that's not directly
// accessible from the logger.
type apiClientWrapper struct {
	response  *telemetry.ResponseBody
	apiClient *client.DatabricksClient
}

func (wrapper *apiClientWrapper) Do(ctx context.Context, method, path string,
	headers map[string]string, request, response any,
	visitors ...func(*http.Request) error) error {

	err := wrapper.apiClient.Do(ctx, method, path, headers, request, response, visitors...)
	wrapper.response = response.(*telemetry.ResponseBody)
	return err
}

func TestAccTelemetryLogger(t *testing.T) {
	ctx, w := acc.WorkspaceTest(t)
	ctx = telemetry.NewContext(ctx)

	// Extend the maximum wait time for the telemetry flush just for this test.
	telemetry.MaxAdditionalWaitTime = 1 * time.Hour
	t.Cleanup(func() {
		telemetry.MaxAdditionalWaitTime = 2 * time.Second
	})

	// Log some events.
	telemetry.Log(ctx, telemetry.FrontendLogEntry{
		DatabricksCliLog: telemetry.DatabricksCliLog{
			CliTestEvent: telemetry.CliTestEvent{
				Name: telemetry.DummyCliEnumValue1,
			},
		},
	})
	telemetry.Log(ctx, telemetry.FrontendLogEntry{
		DatabricksCliLog: telemetry.DatabricksCliLog{
			CliTestEvent: telemetry.CliTestEvent{
				Name: telemetry.DummyCliEnumValue2,
			},
		},
	})

	apiClient, err := client.New(w.W.Config)
	require.NoError(t, err)

	// Flush the events.
	wrapper := &apiClientWrapper{
		apiClient: apiClient,
	}
	telemetry.Flush(ctx, wrapper)

	// Assert that the events were logged.
	assert.Equal(t, telemetry.ResponseBody{
		NumProtoSuccess: 2,
		Errors:          []telemetry.LogError{},
	}, *wrapper.response)
}
