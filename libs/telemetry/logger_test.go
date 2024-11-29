package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDatabricksClient struct {
	numCalls int
	t        *testing.T
}

func (m *mockDatabricksClient) Do(ctx context.Context, method, path string, headers map[string]string, request, response any, visitors ...func(*http.Request) error) error {
	// Block until the fire channel is fired.
	m.numCalls++

	assertRequestPayload := func(reqb RequestBody) {
		expectedProtoLogs := []string{
			fmt.Sprintf("{\"frontend_log_event_id\":\"%s\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE1\"}}}}", root.CommandExecId()),
			fmt.Sprintf("{\"frontend_log_event_id\":\"%s\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}}", root.CommandExecId()),
			fmt.Sprintf("{\"frontend_log_event_id\":\"%s\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}}", root.CommandExecId()),
			fmt.Sprintf("{\"frontend_log_event_id\":\"%s\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE3\"}}}}", root.CommandExecId()),
		}

		// Assert payload matches the expected payload.
		assert.Equal(m.t, expectedProtoLogs, reqb.ProtoLogs)
	}

	switch m.numCalls {
	case 1:
		// The call is successful but not all events are successfully logged.
		assertRequestPayload(request.(RequestBody))
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 3,
		}
	case 2:
		// The call is successful and all events are successfully logged.
		assertRequestPayload(request.(RequestBody))
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 4,
		}
	default:
		panic("unexpected number of calls")
	}

	return nil
}

func TestTelemetryLoggerFlushesEvents(t *testing.T) {
	mockClient := &mockDatabricksClient{
		t: t,
	}

	ctx := NewContext(context.Background())

	for _, v := range []DummyCliEnum{DummyCliEnumValue1, DummyCliEnumValue2, DummyCliEnumValue2, DummyCliEnumValue3} {
		err := Log(ctx, FrontendLogEntry{DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: v},
		}})
		require.NoError(t, err)
	}

	// Flush the events.
	Flush(ctx, mockClient)

	// Assert that the .Do method is called twice, because all logs were not
	// successfully logged in the first call.
	assert.Equal(t, 2, mockClient.numCalls)
}

func TestTelemetryLoggerFlushExitsOnTimeout(t *testing.T) {
	// Set the maximum additional wait time to 0 to ensure that the Flush method times out immediately.
	MaxAdditionalWaitTime = 0
	t.Cleanup(func() {
		MaxAdditionalWaitTime = 2 * time.Second
	})

	mockClient := &mockDatabricksClient{
		t: t,
	}

	ctx := NewContext(context.Background())

	for _, v := range []DummyCliEnum{DummyCliEnumValue1, DummyCliEnumValue2, DummyCliEnumValue2, DummyCliEnumValue3} {
		err := Log(ctx, FrontendLogEntry{DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: v},
		}})
		require.NoError(t, err)
	}

	// Flush the events.
	Flush(ctx, mockClient)

	// Assert that the .Do method is never called since the timeout is set to 0.
	assert.Equal(t, 0, mockClient.numCalls)
}
