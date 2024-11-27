package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDatabricksClient struct {
	numCalls int
}

// TODO: Assert on the request body provided to this method.
func (m *mockDatabricksClient) Do(ctx context.Context, method, path string, headers map[string]string, request, response any, visitors ...func(*http.Request) error) error {
	// For the first two calls, we want to return an error to simulate a server
	// timeout. For the third call, we want to return a successful response.
	m.numCalls++
	switch m.numCalls {
	case 1, 2:
		return fmt.Errorf("server timeout")
	case 3:
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 2,
		}
	case 4:
		return fmt.Errorf("some weird error")
	case 5:
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 3,
		}
	case 6:
		*(response.(*ResponseBody)) = ResponseBody{
			NumProtoSuccess: 4,
		}
	default:
		panic("unexpected number of calls")
	}

	return nil
}

// TODO: Run these tests multiple time to root out race conditions.
func TestTelemetryLoggerPersistentConnectionRetriesOnError(t *testing.T) {
	mockClient := &mockDatabricksClient{}

	ctx, _ := context.WithCancel(context.Background())

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

func TestTelemetryLogger(t *testing.T) {
	mockClient := &mockDatabricksClient{}

	ctx, _ := context.WithCancel(context.Background())

	l, err := NewLogger(ctx, mockClient)
	assert.NoError(t, err)

	// Add three events to be tracked and flushed.
	l.TrackEvent(FrontendLogEntry{
		DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: DummyCliEnumValue1},
		},
	})
	l.TrackEvent(FrontendLogEntry{
		DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: DummyCliEnumValue2},
		},
	})
	l.TrackEvent(FrontendLogEntry{
		DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: DummyCliEnumValue2},
		},
	})
	l.TrackEvent(FrontendLogEntry{
		DatabricksCliLog: DatabricksCliLog{
			CliTestEvent: CliTestEvent{Name: DummyCliEnumValue3},
		},
	})

	// Flush the events.
	l.Flush()

	// Assert that the .Do method was called 6 times. The goroutine should
	// keep on retrying until it sees `numProtoSuccess` equal to 4 since that's
	// the number of events we added.
	assert.Equal(t, 6, mockClient.numCalls)
}
