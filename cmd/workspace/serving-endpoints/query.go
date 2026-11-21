package serving_endpoints

import (
	"bufio"
	"bytes"
	"encoding/json"
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

// queryOverride makes --stream work. The generated command decodes the whole
// response as a single JSON object, so a streamed (server-sent events) response
// always failed with "failed to unmarshal response body". Non-streaming queries
// still run the generated code unchanged.
func queryOverride(queryCmd *cobra.Command, queryReq *serving.QueryEndpointInput) {
	queryCmd.Long += queryStreamHelp
	generatedRunE := queryCmd.RunE
	queryCmd.RunE = func(cmd *cobra.Command, args []string) error {
		stream, err := queryWantsStream(cmd)
		if err != nil || !stream {
			return generatedRunE(cmd, args)
		}

		ctx := cmd.Context()
		if cmd.Flags().Changed("json") {
			jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
			if !ok {
				return generatedRunE(cmd, args)
			}
			diags := jsonFlag.Unmarshal(queryReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				if err := cmdio.RenderDiagnostics(ctx, diags); err != nil {
					return err
				}
			}
		}
		queryReq.Name = args[0]
		queryReq.Stream = true

		w := cmdctx.WorkspaceClient(ctx)
		apiClient, err := client.New(w.Config)
		if err != nil {
			return err
		}
		headers := map[string]string{
			"Accept":       "text/event-stream",
			"Content-Type": "application/json",
		}
		maps.Copy(headers, auth.WorkspaceIDHeaders(w.Config))
		var body io.ReadCloser
		path := fmt.Sprintf("/serving-endpoints/%v/invocations", queryReq.Name)
		err = apiClient.Do(ctx, http.MethodPost, path, headers, nil, queryReq, &body)
		if err != nil {
			return err
		}
		defer body.Close()
		return copyEventStream(cmd.OutOrStdout(), body)
	}
}

// queryWantsStream reports whether --stream is set, or --json sets "stream": true.
func queryWantsStream(cmd *cobra.Command) (bool, error) {
	if stream, err := cmd.Flags().GetBool("stream"); err != nil || stream {
		return stream, err
	}
	if !cmd.Flags().Changed("json") {
		return false, nil
	}
	jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
	if !ok {
		return false, nil
	}
	var body struct {
		Stream bool `json:"stream"`
	}
	// Invalid JSON is reported by the generated command, so ignore it here.
	_ = json.Unmarshal(jsonFlag.Raw(), &body)
	return body.Stream, nil
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
