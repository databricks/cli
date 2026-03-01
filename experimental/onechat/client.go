package onechat

import (
	"context"
	"io"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

const oneChatResponsesPath = "/api/2.0/data-rooms/tools/onechat/responses"

// Streaming timeout: 10 minutes of inactivity before the SDK cancels.
// The OneChat agent can take minutes between SSE events when executing
// multi-step tool calls (search, SQL, etc.).
const streamingTimeoutSeconds = 600

// BuildRequest creates a OneChatRequest for a single-shot question.
func BuildRequest(question, warehouseID string) OneChatRequest {
	req := OneChatRequest{
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
func PostStream(ctx context.Context, cfg *config.Config, req OneChatRequest) (io.ReadCloser, error) {
	// Use a longer inactivity timeout for streaming. The OneChat agent can
	// have multi-minute gaps between SSE events during tool execution.
	cfg.HTTPTimeoutSeconds = streamingTimeoutSeconds

	api, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	var body io.ReadCloser
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "text/event-stream",
	}
	err = api.Do(ctx, "POST", oneChatResponsesPath, headers, nil, req, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
