package agentstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// Response status constants.
const (
	statusCompleted  = "completed"
	statusError      = "error"
	statusIncomplete = "incomplete"
)

// defaultChartWidth is the fixed column budget for terminal charts. Bar and
// line charts cap their own drawing width well below it, so terminal-width
// detection would only matter on very narrow terminals.
const defaultChartWidth = 80

// maxStatusRunes bounds spinner status text: a wrapped status line leaves
// artifacts behind when the spinner erases it.
const maxStatusRunes = 100

// UpdateCLIAdvice tells the user how to recover when the undocumented API
// behind an experimental command has changed or moved: a newer CLI built
// against the current wire format is the only user-side fix.
const UpdateCLIAdvice = "update the Databricks CLI to the latest version (run 'databricks version --check')"

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
// Viz charts are buffered and rendered after all text output so the
// narrative context appears first.
//
// A stream that ends without producing any user-visible answer returns an
// error: the most damaging historical failure mode of this command was a wire
// format change making every item unparseable, which rendered nothing and
// exited 0.
// It returns the response's conversation id (for follow-up turns) and whether
// any output reached stdout.
func RenderText(ctx context.Context, r io.Reader, stdout, stderr io.Writer, adapt AdapterFunc, opts RenderOptions) (string, bool, error) {
	reader := NewSSEReader(r)

	// The spinner degrades to no output on non-interactive terminals, so
	// piped stderr stays free of ANSI escapes. It is closed before anything
	// is written to stdout and recreated on the next thinking update.
	status := cmdio.NewSpinner(ctx)
	stopStatus := func() {
		if status != nil {
			status.Close()
			status = nil
		}
	}
	defer stopStatus()
	updateStatus := func(text string) {
		if status == nil {
			status = cmdio.NewSpinner(ctx)
		}
		status.Update(truncateStatus(text))
	}
	updateStatus("Waiting for response...")

	var vizBuffer []*VizEvent
	var finalResponse string
	var conversationID string
	var rendered strings.Builder // all text shown so far, for final-response dedupe
	wrote := false               // any bytes written to stdout (gates a safe fail-open retry)
	events := 0
	unparsed := 0
	sawDone := false

	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return conversationID, wrote, fmt.Errorf("reading SSE stream: %w", err)
		}
		events++

		for _, se := range adapt(ev.Data) {
			switch se.Kind {
			case EventThinking:
				updateStatus(se.Text)
			case EventText:
				stopStatus()
				if c := renderMarkdown(stdout, se.Text); c != "" {
					wrote = true
					rendered.WriteString(c)
					rendered.WriteString("\n")
				}
			case EventFinalResponse:
				// Buffer the tool-delivered answer; render it after the stream
				// only if no assistant message already showed the answer.
				finalResponse = se.Text
			case EventToolCall:
				if se.ToolCall != nil && opts.ShowSQL && isSQLTool(se.ToolCall.Name) {
					stopStatus()
					renderSQL(stdout, se.ToolCall.Name, se.ToolCall.Arguments)
					wrote = true
				}
			case EventViz:
				if se.Viz != nil {
					vizBuffer = append(vizBuffer, se.Viz)
				}
			case EventError:
				return conversationID, wrote, apiError(se)
			case EventDone:
				sawDone = true
				conversationID = se.ConversationID
				if se.Status != "" && se.Status != statusCompleted {
					return conversationID, wrote, fmt.Errorf("response finished with status %q", se.Status)
				}
			case EventUnparsed:
				unparsed++
			}
		}
	}
	stopStatus()

	// Render the tool-delivered answer unless an assistant message already
	// contained it. Intermediate messages ("Let me check the table first.")
	// must not suppress the actual answer, so this checks content, not just
	// whether any text was rendered. Both sides are compared post-cleanup so
	// a viz reference in one of them cannot defeat the dedupe.
	answered := rendered.Len() > 0
	if final := strings.TrimSpace(cleanMarkdown(finalResponse)); final != "" && !strings.Contains(rendered.String(), final) {
		renderMarkdown(stdout, finalResponse)
		answered = true
		wrote = true
	}

	// Render charts after all text output. A chart that cannot be mapped to
	// the result data degrades to a visible placeholder: the answer text has
	// already had its chart references stripped, so silence here would leave
	// the user staring at "as shown below" with nothing below.
	for _, viz := range vizBuffer {
		wrote = true // every branch writes to stdout, even the placeholder
		if RenderChart(stdout, viz, defaultChartWidth, opts.Color) {
			answered = true
			continue
		}
		if viz.Spec != nil && viz.Spec.Title != "" {
			fmt.Fprintf(stdout, "\n[chart %q could not be rendered in the terminal]\n", viz.Spec.Title)
		} else {
			fmt.Fprintln(stdout, "\n[a chart could not be rendered in the terminal]")
		}
	}

	warnUnparsed(stderr, unparsed)
	if !answered {
		return conversationID, wrote, noAnswerError(events)
	}
	if !sawDone {
		fmt.Fprintln(stderr, "Warning: the stream ended without a completion event; the answer may be incomplete.")
	}
	return conversationID, wrote, nil
}

// RenderJSON accumulates all stream events and emits a single StreamResult
// JSON object. The JSON is written even when the stream fails so callers
// always have structured output to parse; the returned error still makes the
// CLI exit non-zero. Status starts as "incomplete" and is only promoted to
// "completed" by a completion event, so a truncated stream cannot masquerade
// as a successful one.
// It returns the response's conversation id for follow-up turns.
func RenderJSON(r io.Reader, stdout, stderr io.Writer, adapt AdapterFunc) (string, error) {
	reader := NewSSEReader(r)
	result := StreamResult{Status: statusIncomplete}
	var finalResponse string
	var apiErr error
	events := 0
	unparsed := 0

loop:
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Status = statusError
			result.Error = err.Error()
			writeStreamResult(stdout, result)
			return result.ConversationID, fmt.Errorf("reading SSE stream: %w", err)
		}
		events++

		for _, se := range adapt(ev.Data) {
			switch se.Kind {
			case EventThinking, EventViz:
				// Only relevant for text rendering; skip in JSON.
			case EventText:
				result.Text += se.Text
			case EventFinalResponse:
				finalResponse = se.Text
			case EventToolCall:
				if se.ToolCall != nil {
					tc := ToolCall{
						Name:      se.ToolCall.Name,
						Arguments: se.ToolCall.Arguments,
					}
					if isSQLTool(se.ToolCall.Name) {
						tc.SQL, tc.Title = parseSQLArgs(se.ToolCall.Name, se.ToolCall.Arguments)
					}
					result.ToolCalls = append(result.ToolCalls, tc)
				}
			case EventError:
				result.Status = statusError
				result.Error = se.Text
				apiErr = apiError(se)
				break loop
			case EventDone:
				result.ConversationID = se.ConversationID
				if se.Status != "" && se.Status != statusCompleted {
					result.Status = statusError
					result.Error = fmt.Sprintf("response finished with status %q", se.Status)
					apiErr = fmt.Errorf("response finished with status %q", se.Status)
					break loop
				}
				result.Status = statusCompleted
			case EventUnparsed:
				unparsed++
			}
		}
	}

	// Mirror the text renderer: append the tool-delivered answer unless the
	// message text already contains it. The comparison is post-cleanup so a
	// viz reference in one of them cannot defeat the dedupe; the JSON itself
	// keeps the raw markdown.
	if final := strings.TrimSpace(cleanMarkdown(finalResponse)); final != "" && !strings.Contains(cleanMarkdown(result.Text), final) {
		if result.Text != "" {
			result.Text += "\n\n"
		}
		result.Text += finalResponse
	}

	if apiErr == nil {
		if result.Text == "" && len(result.ToolCalls) == 0 {
			result.Status = statusError
			apiErr = noAnswerError(events)
		} else if result.Status == statusIncomplete {
			// Keep status "incomplete": an answer was produced, the server
			// just never confirmed completion.
			apiErr = errors.New("the stream ended without a completion event; the response may be incomplete")
		}
		if apiErr != nil {
			result.Error = apiErr.Error()
		}
	}

	warnUnparsed(stderr, unparsed)
	writeStreamResult(stdout, result)
	return result.ConversationID, apiErr
}

// writeStreamResult encodes the result to w. Encoding errors are reported on
// the same path as the result itself, so they are intentionally not returned:
// if w is broken there is nowhere left to report to.
func writeStreamResult(w io.Writer, result StreamResult) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

// apiError converts an EventError into the error returned to the user. The
// renderers do not print it themselves; the root command prints returned
// errors, and printing in both places showed every API error twice.
func apiError(se StreamEvent) error {
	if se.ErrorCode == "" {
		return fmt.Errorf("API error: %s", se.Text)
	}
	return fmt.Errorf("API error: %s: %s", se.ErrorCode, se.Text)
}

// noAnswerError reports a stream that ended without any user-visible answer.
// Short of a server bug, this means the wire format drifted away from what
// this build understands, so the message leads with the CLI update advice.
func noAnswerError(events int) error {
	return fmt.Errorf("the stream ended without an answer (received %d events); the API may have changed: %s, or re-run with --raw to inspect the raw stream", events, UpdateCLIAdvice)
}

// warnUnparsed reports events the adapter recognized but could not decode.
// These are dropped from rendering, and a wire format drift that drops
// everything must be visible rather than an empty success.
func warnUnparsed(stderr io.Writer, unparsed int) {
	if unparsed > 0 {
		fmt.Fprintf(stderr, "Warning: %d stream event(s) could not be parsed and were ignored; the API may have changed: %s, or re-run with --raw to inspect the raw stream.\n", unparsed, UpdateCLIAdvice)
	}
}

// truncateStatus bounds status text to one spinner line, collapsing newlines
// so multi-sentence thoughts do not break the line-erase protocol.
func truncateStatus(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	runes := []rune(text)
	if len(runes) <= maxStatusRunes {
		return text
	}
	return string(runes[:maxStatusRunes-3]) + "..."
}
