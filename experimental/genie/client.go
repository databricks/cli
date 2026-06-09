package genie

import (
	"context"
	"io"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

// genieResponsesPath is the backend route. The server-side tool is still named
// "onechat" even though the CLI command is "genie", so the path keeps that name.
const genieResponsesPath = "/api/2.0/data-rooms/tools/onechat/responses"

// StreamingTimeoutSeconds is how long the SDK waits between body reads
// before canceling the stream. The Genie agent can take minutes between SSE
// events when executing multi-step tool calls (search, SQL, etc.), so this
// is much higher than the SDK's 60s default.
const StreamingTimeoutSeconds = 600

// BuildRequest creates a GenieRequest for a single-shot question. An empty
// warehouseID is omitted from the wire format and the backend auto-resolves.
func BuildRequest(question, warehouseID string) GenieRequest {
	return GenieRequest{
		Input: []InputItem{
			{
				Type: "message",
				Role: "user",
				Content: []ContentItem{
					{Type: "input_text", Text: question},
				},
			},
		},
		WarehouseID: warehouseID,
	}
}

// PostStream sends the request and returns the raw SSE response body.
// The caller must close the returned ReadCloser.
//
// The inactivity timeout is raised on cfg in place: config.Config embeds a
// mutex, so copying it is a go vet copylocks violation, and the SDK offers no
// per-request timeout. The command makes no other requests from this config.
func PostStream(ctx context.Context, cfg *config.Config, req GenieRequest) (io.ReadCloser, error) {
	cfg.HTTPTimeoutSeconds = StreamingTimeoutSeconds

	api, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	var body io.ReadCloser
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "text/event-stream",
	}
	err = api.Do(ctx, "POST", genieResponsesPath, headers, nil, req, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
