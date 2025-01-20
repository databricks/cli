package telemetry_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/google/uuid"
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
	visitors ...func(*http.Request) error,
) error {
	err := wrapper.apiClient.Do(ctx, method, path, headers, request, response, visitors...)
	wrapper.response = response.(*telemetry.ResponseBody)
	return err
}

func TestTelemetryLogger(t *testing.T) {
	events := []telemetry.DatabricksCliLog{
		{
			CliTestEvent: &protos.CliTestEvent{
				Name: protos.DummyCliEnumValue1,
			},
		},
		{
			BundleInitEvent: &protos.BundleInitEvent{
				Uuid:         uuid.New().String(),
				TemplateName: "abc",
				TemplateEnumArgs: []protos.BundleInitTemplateEnumArg{
					{
						Key:   "a",
						Value: "b",
					},
					{
						Key:   "c",
						Value: "d",
					},
				},
			},
		},
	}

	assert.Equal(t, len(events), reflect.TypeOf(telemetry.DatabricksCliLog{}).NumField(),
		"Number of events should match the number of fields in DatabricksCliLog. Please add a new event to this test.")

	ctx, w := acc.WorkspaceTest(t)
	ctx = telemetry.WithDefaultLogger(ctx)

	// Extend the maximum wait time for the telemetry flush just for this test.
	oldV := telemetry.MaxAdditionalWaitTime
	telemetry.MaxAdditionalWaitTime = 1 * time.Hour
	t.Cleanup(func() {
		telemetry.MaxAdditionalWaitTime = oldV
	})

	for _, event := range events {
		telemetry.Log(ctx, event)
	}

	apiClient, err := client.New(w.W.Config)
	require.NoError(t, err)

	// Flush the events.
	wrapper := &apiClientWrapper{
		apiClient: apiClient,
	}
	telemetry.Flush(ctx, wrapper)

	// Assert that the events were logged.
	assert.Equal(t, telemetry.ResponseBody{
		NumProtoSuccess: int64(len(events)),
		Errors:          []telemetry.LogError{},
	}, *wrapper.response)
}
