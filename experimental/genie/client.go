package genie

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"

	"github.com/databricks/cli/experimental/genie/agentstream"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/apierr"
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
	maps.Copy(headers, auth.WorkspaceIDHeaders(cfg))
	err = api.Do(ctx, "POST", genieResponsesPath, headers, nil, req, &body)
	// The route is fixed and carries no resource IDs, so a 404 normally means
	// the endpoint itself is gone: the backend route is undocumented and can
	// move or be disabled between Databricks releases (a removed route returns
	// 404 ENDPOINT_NOT_FOUND, "No API found for ...", which the SDK maps to
	// plain ErrNotFound). A 404 RESOURCE_DOES_NOT_EXIST is excluded: it refers
	// to something the request named (e.g. the warehouse) and must keep the
	// backend's own message instead of blaming the endpoint.
	if errors.Is(err, apierr.ErrNotFound) && !errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return nil, fmt.Errorf("the Genie API is not available on this workspace: %w; the endpoint may have moved since this CLI release: %s", err, agentstream.UpdateCLIAdvice)
	}
	// A request body the backend cannot interpret (e.g. after its expected
	// request shape changed) surfaces as a 500 INTERNAL_ERROR with an empty
	// message (observed live), leaving the user a blank error. Transient
	// backend faults share the status code, hence the hedged advice.
	if errors.Is(err, apierr.ErrInternalError) {
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.Message == "" {
			// An empty message would render as "request: ;" mid-sentence, so
			// the observed no-details shape gets its own wording. The %w
			// keeps the error chain and renders as nothing.
			return nil, fmt.Errorf("the Genie backend could not process the request (500 with no details)%w; if this keeps happening, the request format may have changed since this CLI release: %s", err, agentstream.UpdateCLIAdvice)
		}
		return nil, fmt.Errorf("the Genie backend could not process the request: %w; if this keeps happening, the request format may have changed since this CLI release: %s", err, agentstream.UpdateCLIAdvice)
	}
	if err != nil {
		return nil, err
	}
	return body, nil
}
