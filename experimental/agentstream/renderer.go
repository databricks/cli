package agentstream

import (
	"encoding/json"
	"fmt"
	"io"
)

// RenderDebug prints every raw SSE data line to w as-is.
func RenderDebug(r io.Reader, w io.Writer) error {
	reader := NewSSEReader(r)
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading SSE stream: %w", err)
		}
		fmt.Fprintln(w, ev.Data)
	}
}

// RenderText renders the SSE stream as human-readable text.
// The adapt function converts each raw SSE payload into StreamEvents.
func RenderText(r io.Reader, stdout, stderr io.Writer, adapt AdapterFunc) error {
	reader := NewSSEReader(r)
	status := &statusLine{w: stderr}
	status.update("Waiting for response...")

	for {
		ev, err := reader.Next()
		if err == io.EOF {
			status.clear()
			return nil
		}
		if err != nil {
			status.clear()
			return fmt.Errorf("reading SSE stream: %w", err)
		}

		events := adapt(ev.Data)
		for _, se := range events {
			switch se.Kind {
			case EventThinking:
				status.update(se.Text)
			case EventText:
				status.clear()
				renderMarkdown(stdout, se.Text)
			case EventToolCall:
				if se.ToolCall != nil && se.ToolCall.Name == "execute_sql" {
					status.clear()
					renderSQL(stdout, se.ToolCall.Arguments)
				}
			case EventError:
				status.clear()
				fmt.Fprintf(stderr, "Error: %s\n", se.Text)
				return fmt.Errorf("API error: %s: %s", se.ErrorCode, se.Text)
			case EventDone:
				status.clear()
				if se.Status != "" && se.Status != "completed" {
					return fmt.Errorf("response finished with status %q", se.Status)
				}
			}
		}
	}
}

// RenderJSON accumulates all stream events and emits a single StreamResult JSON object.
// Returns an error on API errors so the CLI exits non-zero.
func RenderJSON(r io.Reader, w io.Writer, adapt AdapterFunc) error {
	reader := NewSSEReader(r)
	result := StreamResult{Status: "completed"}
	var apiErr error

	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading SSE stream: %w", err)
		}

		events := adapt(ev.Data)
		for _, se := range events {
			switch se.Kind {
			case EventThinking:
				// Thinking events are only relevant for text rendering; skip in JSON.
			case EventText:
				result.Text = se.Text
			case EventToolCall:
				if se.ToolCall != nil {
					tc := ToolCall{
						Name:      se.ToolCall.Name,
						Arguments: se.ToolCall.Arguments,
					}
					if se.ToolCall.Name == "execute_sql" {
						tc.SQL, tc.Title = parseSQLArgs(se.ToolCall.Arguments)
					}
					result.ToolCalls = append(result.ToolCalls, tc)
				}
			case EventError:
				result.Status = "error"
				result.Text = se.Text
				apiErr = fmt.Errorf("API error: %s: %s", se.ErrorCode, se.Text)
			case EventDone:
				if se.Status != "" && se.Status != "completed" {
					result.Status = "error"
					apiErr = fmt.Errorf("response finished with status %q", se.Status)
				}
			}
		}

		// Stop accumulating after an error.
		if apiErr != nil {
			break
		}
	}

	// Always write the JSON so agents can parse the structured output.
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}

	return apiErr
}
