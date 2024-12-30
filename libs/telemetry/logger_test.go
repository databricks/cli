package telemetry

import (
	"context"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/telemetry/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
			"{\"frontend_log_event_id\":\"0194fdc2-fa2f-4cc0-81d3-ff12045b73c8\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE1\"}}}}",
			"{\"frontend_log_event_id\":\"6e4ff95f-f662-45ee-a82a-bdf44a2d0b75\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}}",
			"{\"frontend_log_event_id\":\"fb180daf-48a7-4ee0-b10d-394651850fd4\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}}",
			"{\"frontend_log_event_id\":\"a178892e-e285-4ce1-9114-55780875d64e\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE3\"}}}}",
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

	// Set the random number generator to a fixed seed to ensure that the UUIDs are deterministic.
	uuid.SetRand(rand.New(rand.NewSource(0)))
	t.Cleanup(func() {
		uuid.SetRand(nil)
	})

	ctx := ContextWithLogger(context.Background())

	for _, v := range []events.DummyCliEnum{events.DummyCliEnumValue1, events.DummyCliEnumValue2, events.DummyCliEnumValue2, events.DummyCliEnumValue3} {
		Log(ctx, DatabricksCliLog{
			CliTestEvent: &events.CliTestEvent{Name: v},
		})
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

	// Set the random number generator to a fixed seed to ensure that the UUIDs are deterministic.
	uuid.SetRand(rand.New(rand.NewSource(0)))
	t.Cleanup(func() {
		uuid.SetRand(nil)
	})

	ctx := ContextWithLogger(context.Background())

	for _, v := range []events.DummyCliEnum{events.DummyCliEnumValue1, events.DummyCliEnumValue2, events.DummyCliEnumValue2, events.DummyCliEnumValue3} {
		Log(ctx, DatabricksCliLog{
			CliTestEvent: &events.CliTestEvent{Name: v},
		})
	}

	// Flush the events.
	Flush(ctx, mockClient)

	// Assert that the .Do method is never called since the timeout is set to 0.
	assert.Equal(t, 0, mockClient.numCalls)
}
