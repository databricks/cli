package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockDatabricksClient struct {
	numCalls int

	t *testing.T
}

func (m *mockDatabricksClient) Do(ctx context.Context, method, path string, headers map[string]string, request, response any, visitors ...func(*http.Request) error) error {
	m.numCalls++

	assertRequestPayload := func() {
		expectedProtoLogs := []string{
			"{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE1\"}}}",
			"{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}",
			"{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}",
			"{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE3\"}}}",
		}

		// Assert payload matches the expected payload.
		assert.Equal(m.t, expectedProtoLogs, request.(RequestBody).ProtoLogs)
	}

	switch m.numCalls {
	case 1, 2:
		// Assert that the request is of type *io.PipeReader, which implies that
		// the request is not coming from the main thread.
		assert.IsType(m.t, &io.PipeReader{}, request)

		// For the first two calls, we want to return an error to simulate a server
		// timeout.
		return fmt.Errorf("server timeout")
	case 3:
		// Assert that the request is of type *io.PipeReader, which implies that
		// the request is not coming from the main thread.
		assert.IsType(m.t, &io.PipeReader{}, request)

		// The call is successful but not all events are successfully logged.
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 2,
		}
	case 4:
		// Assert that the request is of type RequestBody, which implies that the
		// request is coming from the main thread.
		assertRequestPayload()
		return fmt.Errorf("some weird error")
	case 5:
		// The call is successful but not all events are successfully logged.
		assertRequestPayload()
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 3,
		}
	case 6:
		// The call is successful and all events are successfully logged.
		assertRequestPayload()
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 4,
		}
	default:
		panic("unexpected number of calls")
	}

	return nil
}

// We run each of the unit tests multiple times to root out any race conditions
// that may exist.
func TestTelemetryLogger(t *testing.T) {
	for i := 0; i < 5000; i++ {
		t.Run("testPersistentConnectionRetriesOnError", testPersistentConnectionRetriesOnError)
		t.Run("testFlush", testFlush)
		t.Run("testFlushRespectsTimeout", testFlushRespectsTimeout)
	}
}

func testPersistentConnectionRetriesOnError(t *testing.T) {
	mockClient := &mockDatabricksClient{
		t: t,
	}

	ctx := context.Background()

	l, err := NewLogger(ctx, mockClient)
	assert.NoError(t, err)

	// Wait for the persistent connection go-routine to exit.
	resp := <-l.respChannel

	// Assert that the .Do method was called 3 times. The goroutine should
	// return on the first successful response.
	assert.Equal(t, 3, mockClient.numCalls)

	// Assert the value of the response body.
	assert.Equal(t, int64(2), resp.NumProtoSuccess)
}

func testFlush(t *testing.T) {
	mockClient := &mockDatabricksClient{
		t: t,
	}

	ctx := context.Background()

	l, err := NewLogger(ctx, mockClient)
	assert.NoError(t, err)

	// Set the maximum additional wait time to 1 hour to ensure that the
	// the Flush method does not timeout in the test run.
	MaxAdditionalWaitTime = 1 * time.Hour
	t.Cleanup(func() {
		MaxAdditionalWaitTime = 1 * time.Second
	})

	// Add four events to be tracked and flushed.
	for _, v := range []DummyCliEnum{DummyCliEnumValue1, DummyCliEnumValue2, DummyCliEnumValue2, DummyCliEnumValue3} {
		l.TrackEvent(FrontendLogEntry{
			DatabricksCliLog: DatabricksCliLog{
				CliTestEvent: CliTestEvent{Name: v},
			},
		})
	}

	// Flush the events.
	l.Flush()

	// Assert that the .Do method was called 6 times. The goroutine should
	// keep on retrying until it sees `numProtoSuccess` equal to 4 since that's
	// the number of events we added.
	assert.Equal(t, 6, mockClient.numCalls)
}

func testFlushRespectsTimeout(t *testing.T) {
	mockClient := &mockDatabricksClient{
		t: t,
	}

	ctx := context.Background()

	l, err := NewLogger(ctx, mockClient)
	assert.NoError(t, err)

	// Set the timer to 0 to ensure that the Flush method times out immediately.
	MaxAdditionalWaitTime = 0 * time.Hour
	t.Cleanup(func() {
		MaxAdditionalWaitTime = 1 * time.Second
	})

	// Add four events to be tracked and flushed.
	for _, v := range []DummyCliEnum{DummyCliEnumValue1, DummyCliEnumValue2, DummyCliEnumValue2, DummyCliEnumValue3} {
		l.TrackEvent(FrontendLogEntry{
			DatabricksCliLog: DatabricksCliLog{
				CliTestEvent: CliTestEvent{Name: v},
			},
		})
	}

	// Flush the events.
	l.Flush()

	// Assert that the .Do method was called less than or equal to 3 times. Since
	// the timeout is set to 0, only the calls from the parallel go-routine should
	// be made. The main thread should not make any calls.
	assert.LessOrEqual(t, mockClient.numCalls, 3)
}
