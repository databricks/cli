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

// Streaming timeout: 10 minutes of inactivity before the SDK cancels.
// The Genie agent can take minutes between SSE events when executing
// multi-step tool calls (search, SQL, etc.).
const streamingTimeoutSeconds = 600

// BuildRequest creates a GenieRequest for a single-shot question.
func BuildRequest(question, warehouseID string) GenieRequest {
	req := GenieRequest{
		Input: []InputItem{
			{
				Type: "message",
				Role: "user",
				Content: []ContentItem{
					{Type: "input_text", Text: question},
				},
			},
		},
	}
	if warehouseID != "" {
		req.WarehouseID = warehouseID
	}
	return req
}

// PostStream sends the request and returns the raw SSE response body.
// The caller must close the returned ReadCloser.
func PostStream(ctx context.Context, cfg *config.Config, req GenieRequest) (io.ReadCloser, error) {
	streamCfg := *cfg
	streamCfg.HTTPTimeoutSeconds = streamingTimeoutSeconds

	api, err := client.New(&streamCfg)
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
