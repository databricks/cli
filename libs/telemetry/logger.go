package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/google/uuid"
)

// Interface abstraction created to mock out the Databricks client for testing.
type DatabricksApiClient interface {
	Do(ctx context.Context, method, path string,
		headers map[string]string, request, response any,
		visitors ...func(*http.Request) error) error
}

func Log(ctx context.Context, event DatabricksCliLog) error {
	l := fromContext(ctx)

	FrontendLog := FrontendLog{
		// The telemetry endpoint deduplicates logs based on the FrontendLogEventID.
		// This it's important to generate a unique ID for each log event.
		FrontendLogEventID: uuid.New().String(),
		Entry: FrontendLogEntry{
			DatabricksCliLog: event,
		},
	}

	protoLog, err := json.Marshal(FrontendLog)
	if err != nil {
		return fmt.Errorf("error marshalling the telemetry event: %v", err)
	}

	l.protoLogs = append(l.protoLogs, string(protoLog))
	return nil
}

type logger struct {
	protoLogs []string
}

// This function is meant to be only to be used in tests to introspect the telemetry logs
// that have been logged so far.
func GetLogs(ctx context.Context) ([]FrontendLog, error) {
	l := fromContext(ctx)
	res := []FrontendLog{}

	for _, log := range l.protoLogs {
		frontendLog := FrontendLog{}
		err := json.Unmarshal([]byte(log), &frontendLog)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling the telemetry event: %v", err)
		}
		res = append(res, frontendLog)
	}
	return res, nil
}

// Maximum additional time to wait for the telemetry event to flush. We expect the flush
// method to be called when the CLI command is about to exist, so this caps the maximum
// additional time the user will experience because of us logging CLI telemetry.
var MaxAdditionalWaitTime = 2 * time.Second

// We make the API call to the /telemetry-ext endpoint to log the CLI telemetry events
// right about as the CLI command is about to exit. The API endpoint can handle
// payloads with ~1000 events easily. Thus we log all the events at once instead of
// batching the logs across multiple API calls.
func Flush(ctx context.Context, apiClient DatabricksApiClient) {
	// Set a maximum time to wait for the telemetry event to flush.
	ctx, cancel := context.WithTimeout(ctx, MaxAdditionalWaitTime)
	defer cancel()
	l := fromContext(ctx)

	if len(l.protoLogs) == 0 {
		log.Debugf(ctx, "No telemetry events to flush")
		return
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
			ProtoLogs:  l.protoLogs,

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
		if len(l.protoLogs) > int(resp.NumProtoSuccess) {
			log.Debugf(ctx, "Not all logs were successfully sent. Retrying...")
			continue
		}

		// All logs were successfully sent. We can exit the function.
		log.Debugf(ctx, "Successfully flushed telemetry events")
		return
	}
}
