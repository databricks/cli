package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/client"
)

// Interface abstraction created to mock out the Databricks client for testing.
type databricksClient interface {
	Do(ctx context.Context, method, path string,
		headers map[string]string, request, response any,
		visitors ...func(*http.Request) error) error
}

type logger struct {
	respChannel chan *ResponseBody

	apiClient databricksClient

	// TODO: Appropriately name this field.
	w *io.PipeWriter

	// TODO: wrap this in a mutex since it'll be concurrently accessed.
	protoLogs []string
}

// TODO: Test that both sever and client side timeouts will spawn a new goroutine
// TODO: Add comment here about the warm pool stuff.
// TODO: Talk about why this request is made in a separate goroutine in the
// background while the other requests are made in the main thread (since the
// retry operations can be blocking).
// TODO: The main thread also needs to appropriately communicate with this
// thread.
//
//	TODO: Add an integration test for this functionality as well.

// spawnTelemetryConnection will spawn a new TCP connection to the telemetry
// endpoint and keep it alive until the main CLI thread is alive.
//
// Both the Databricks Go SDK client and Databricks control plane servers typically
// timeout after 60 seconds. Thus if we see any error from the API client we'll
// simply retry the request to establish a new TCP connection.
//
// The intent of this function is to reduce the RTT for the HTTP request to the telemetry
// endpoint since underneath the hood the Go standard library http client will establish
// the connection but will be blocked on reading the request body until we write
// to the corresponding pipe writer for the request body pipe reader.
//
// Benchmarks suggest this reduces the RTT from ~700 ms to ~200 ms.
func (l *logger) spawnTelemetryConnection(ctx context.Context, r *io.PipeReader) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Proceed
		}

		resp := &ResponseBody{}

		// This API request will exchange TCP/TLS headers with the server but would
		// be blocked on sending over the request body until we write to the
		// corresponding writer for the request body reader.
		err := l.apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, r, resp)

		// The TCP connection can timeout while it waits for the CLI to send over
		// the request body. It could be either due to the client which has a
		// default timeout of 60 seconds or a server side timeout with a status code
		// of 408.
		//
		// Thus as long as the main CLI thread is alive we'll simply keep trying again
		// and establish a new TCL connection.
		//
		// TODO: Verify whether the server side timeout is 408 for the telemetry API
		// TODO: Check where the telemetry API even has a server side timeout.
		if err == nil {
			l.respChannel <- resp
			return
		}
	}

}

// TODO: Log warning or errors when any of these telemetry requests fail.
// TODO: Figure out how to create or use an existing general purpose http mocking
// library to unit test this functionality out.
// TODO: Add the concurrency functionality to make the API call from a separate thread.
// TODO: Add a cap for the maximum local timeout how long we'll wait for the telemetry
// event to flush.
// TODO: Do I need to customize exponential backoff for this? Since we don't want to
// wait too long in between retries.
// TODO: test that this client works for long running API calls.
func NewLogger(ctx context.Context, apiClient databricksClient) (*logger, error) {
	if apiClient == nil {
		var err error
		apiClient, err = client.New(root.WorkspaceClient(ctx).Config)
		if err != nil {
			return nil, fmt.Errorf("error creating telemetry client: %v", err)
		}
	}

	r, w := io.Pipe()

	l := &logger{
		protoLogs:   []string{},
		apiClient:   apiClient,
		w:           w,
		respChannel: make(chan *ResponseBody, 1),
	}

	go func() {
		l.spawnTelemetryConnection(ctx, r)
	}()

	return l, nil
}

// TODO: Add unit test for this and verify that the quotes are retained.
func (l *logger) TrackEvent(event FrontendLogEntry) {
	protoLog, err := json.Marshal(event)
	if err != nil {
		return
	}

	l.protoLogs = append(l.protoLogs, string(protoLog))
}

// Maximum additional time to wait for the telemetry event to flush. We expect the flush
// method to be called when the CLI command is about to exist, so this caps the maximum
// additional time the user will experience because of us logging CLI telemetry.
var MaxAdditionalWaitTime = 1 * time.Second

// TODO: Talk about why we make only one API call at the end. It's because the
// size limit on the payload is pretty high: ~1000 events.
func (l *logger) Flush(ctx context.Context) {
	// Set a maximum time to wait for the telemetry event to flush.
	ctx, _ = context.WithTimeout(ctx, MaxAdditionalWaitTime)
	var resp *ResponseBody

	reqb := RequestBody{
		UploadTime: time.Now().Unix(),
		ProtoLogs:  l.protoLogs,
	}

	// Finally write to the pipe writer to unblock the API request.
	b, err := json.Marshal(reqb)
	if err != nil {
		log.Debugf(ctx, "Error marshalling telemetry logs: %v", err)
		return
	}
	_, err = l.w.Write(b)
	if err != nil {
		log.Debugf(ctx, "Error writing to telemetry pipe: %v", err)
		return
	}

	select {
	case <-ctx.Done():
		log.Debugf(ctx, "Timed out before flushing telemetry events")
		return
	case resp = <-l.respChannel:
		// The persistent TCP connection we create finally returned a response
		// from the /telemetry-ext endpoint. We can now start processing the
		// response in the main thread.
	}

	for {
		select {
		case <-ctx.Done():
			log.Debugf(ctx, "Timed out before flushing telemetry events")
			return
		default:
			// Proceed
		}

		// All logs were successfully sent. No need to retry.
		if len(l.protoLogs) <= int(resp.NumProtoSuccess) && len(resp.Errors) == 0 {
			return
		}

		// Some or all logs were not successfully sent. We'll retry and log everything
		// again.
		//
		// Note: This will result in server side duplications but that's fine since
		// we can always deduplicate in the data pipeline itself.
		l.apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, RequestBody{
			UploadTime: time.Now().Unix(),
			ProtoLogs:  l.protoLogs,
		}, resp)
	}
}
