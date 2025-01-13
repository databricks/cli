package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/events"
	"github.com/google/uuid"
)

// Interface abstraction created to mock out the Databricks client for testing.
type DatabricksApiClient interface {
	Do(ctx context.Context, method, path string,
		headers map[string]string, request, response any,
		visitors ...func(*http.Request) error) error
}

type Logger interface {
	// Record a telemetry event, to be flushed later.
	Log(ctx context.Context, event DatabricksCliLog)

	// Flush all the telemetry events that have been logged so far. We expect
	// this to be called once per CLI command for the default logger.
	Flush(ctx context.Context, executionContext *events.ExecutionContext, apiClient DatabricksApiClient)

	// This function is meant to be only to be used in tests to introspect
	// the telemetry logs that have been logged so far.
	Introspect() []DatabricksCliLog
}

type defaultLogger struct {
	logs []*FrontendLog
}

func (l *defaultLogger) Log(ctx context.Context, event DatabricksCliLog) {
	if l.logs == nil {
		l.logs = make([]*FrontendLog, 0)
	}
	l.logs = append(l.logs, &FrontendLog{
		// The telemetry endpoint deduplicates logs based on the FrontendLogEventID.
		// This it's important to generate a unique ID for each log event.
		FrontendLogEventID: uuid.New().String(),
		Entry: FrontendLogEntry{
			DatabricksCliLog: event,
		},
	})
}

// Maximum additional time to wait for the telemetry event to flush. We expect the flush
// method to be called when the CLI command is about to exit, so this caps the maximum
// additional time the user will experience because of us logging CLI telemetry.
var MaxAdditionalWaitTime = 3 * time.Second

// We make the API call to the /telemetry-ext endpoint to log the CLI telemetry events
// right about as the CLI command is about to exit. The API endpoint can handle
// payloads with ~1000 events easily. Thus we log all the events at once instead of
// batching the logs across multiple API calls.
func (l *defaultLogger) Flush(ctx context.Context, executionContext *events.ExecutionContext, apiClient DatabricksApiClient) {
	// Set a maximum time to wait for the telemetry event to flush.
	ctx, cancel := context.WithTimeout(ctx, MaxAdditionalWaitTime)
	defer cancel()

	if executionContext != nil {
		for _, event := range l.logs {
			event.Entry.DatabricksCliLog.ExecutionContext = *executionContext
		}
	}

	if len(l.logs) == 0 {
		log.Debugf(ctx, "No telemetry events to flush")
		return
	}

	var protoLogs []string
	for _, event := range l.logs {
		s, err := json.Marshal(event)
		if err != nil {
			log.Debugf(ctx, "Error marshalling the telemetry event %v: %v", event, err)
			continue
		}

		protoLogs = append(protoLogs, string(s))
	}

	resp := &ResponseBody{}
	for {
		select {
		case <-ctx.Done():
			log.Debugf(ctx, "Timed out before flushing telemetry events")
			return
		default:
			// Proceed
		}

		// Log the CLI telemetry events.
		err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, RequestBody{
			UploadTime: time.Now().Unix(),
			ProtoLogs:  protoLogs,

			// A bug in the telemetry API requires us to send an empty items array.
			// Otherwise we get an opaque 500 internal server error.
			Items: []string{},
		}, resp)
		if err != nil {
			// The SDK automatically performs retries for 429s and 503s. Thus if we
			// see an error here, do not retry logging the telemetry.
			log.Debugf(ctx, "Error making the API request to /telemetry-ext: %v", err)
			return
		}
		// If not all the logs were successfully sent, we'll retry and log everything
		// again.
		//
		// Note: This will result in server side duplications but that's fine since
		// we can always deduplicate in the data pipeline itself.
		if len(l.logs) > int(resp.NumProtoSuccess) {
			log.Debugf(ctx, "Not all logs were successfully sent. Retrying...")
			continue
		}

		// All logs were successfully sent. We can exit the function.
		log.Debugf(ctx, "Successfully flushed telemetry events")
		return
	}
}

func (l *defaultLogger) Introspect() []DatabricksCliLog {
	panic("not implemented")
}

func Log(ctx context.Context, event DatabricksCliLog) {
	l := fromContext(ctx)
	l.Log(ctx, event)
}

func Flush(ctx context.Context, executionContext *events.ExecutionContext, apiClient DatabricksApiClient) {
	l := fromContext(ctx)
	l.Flush(ctx, executionContext, apiClient)
}

func Introspect(ctx context.Context) []DatabricksCliLog {
	l := fromContext(ctx)
	return l.Introspect()
}
