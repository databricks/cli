package serving_endpoints

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

const (
	sseDataPrefix = "data:"
	sseDone       = "[DONE]"
)

const queryStreamHelp = `

  With --stream (or "stream": true in --json), the response is printed as it
  arrives: one JSON chunk per line, without the server-sent event framing.
  For a chat endpoint, the reply text of each chunk is in
  .choices[0].delta.content.`

// queryOverride sends the query itself instead of through the SDK's typed
// Query call, for two reasons:
//   - The SDK decodes the whole response as one JSON object, so a streamed
//     (server-sent events) response always failed with "failed to unmarshal
//     response body". Streamed chunks are now printed one per line.
//   - The SDK's response type only has some of the fields endpoints return,
//     and drops the rest without a trace (for example choices[].finish_reason,
//     tool calls, or an agent's custom outputs). The output is still the typed
//     response, so it doesn't change, but a warning on stderr now names the
//     fields it left out.
func queryOverride(queryCmd *cobra.Command, queryReq *serving.QueryEndpointInput) {
	queryCmd.Long += queryStreamHelp
	queryCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := unmarshalQueryJSON(cmd, queryReq); err != nil {
			return err
		}
		queryReq.Name = args[0]

		accept := "application/json"
		if queryReq.Stream {
			accept = "text/event-stream"
		}
		body, err := postInvocations(ctx, queryReq, accept)
		if err != nil {
			return err
		}
		defer body.Close()
		if queryReq.Stream {
			return copyEventStream(cmd.OutOrStdout(), body)
		}

		raw, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		var response serving.QueryEndpointResponse
		if err := json.Unmarshal(raw, &response); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
		if err := warnDroppedFields(ctx, queryReq.Name, raw, &response); err != nil {
			return err
		}
		return cmdio.Render(ctx, &response)
	}
}

// unmarshalQueryJSON applies --json the same way the generated command does.
func unmarshalQueryJSON(cmd *cobra.Command, queryReq *serving.QueryEndpointInput) error {
	if !cmd.Flags().Changed("json") {
		return nil
	}
	jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
	if !ok {
		return errors.New("unexpected type for --json flag")
	}
	diags := jsonFlag.Unmarshal(queryReq)
	if diags.HasError() {
		return diags.Error()
	}
	if len(diags) > 0 {
		return cmdio.RenderDiagnostics(cmd.Context(), diags)
	}
	return nil
}

// postInvocations sends the query with the same path, headers and body as the
// SDK's Query call, and returns the response body unread.
func postInvocations(ctx context.Context, queryReq *serving.QueryEndpointInput, accept string) (io.ReadCloser, error) {
	w := cmdctx.WorkspaceClient(ctx)
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Accept":       accept,
		"Content-Type": "application/json",
	}
	maps.Copy(headers, auth.WorkspaceIDHeaders(w.Config))
	var body io.ReadCloser
	path := fmt.Sprintf("/serving-endpoints/%v/invocations", queryReq.Name)
	err = apiClient.Do(ctx, http.MethodPost, path, headers, nil, queryReq, &body)
	return body, err
}

// copyEventStream writes the data payload of each server-sent event as one
// line, and stops at the "[DONE]" sentinel. Lines that aren't SSE framing (for
// example a plain JSON body from a model that ignores "stream") are copied as is.
func copyEventStream(out io.Writer, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	// Chunks can be large (for example tool call arguments), so allow long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if payload, ok := bytes.CutPrefix(line, []byte(sseDataPrefix)); ok {
			payload = bytes.TrimSpace(payload)
			if string(payload) == sseDone {
				return nil
			}
			line = payload
		} else if isSSEFraming(line) {
			continue
		}
		if _, err := fmt.Fprintf(out, "%s\n", line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// isSSEFraming reports whether line is an SSE blank line, comment or non-data field.
func isSSEFraming(line []byte) bool {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] == ':' {
		return true
	}
	for _, field := range []string{"event:", "id:", "retry:"} {
		if bytes.HasPrefix(line, []byte(field)) {
			return true
		}
	}
	return false
}

func init() {
	queryOverrides = append(queryOverrides, queryOverride)
}
