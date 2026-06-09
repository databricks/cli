package agentstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSSE builds an SSE stream from data payloads.
func fakeSSE(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("data: ")
		b.WriteString(p)
		b.WriteString("\n\n")
	}
	return b.String()
}

// testAdapter returns an AdapterFunc that maps data payloads to predefined events.
func testAdapter(mapping map[string][]StreamEvent) AdapterFunc {
	return func(data string) []StreamEvent {
		return mapping[data]
	}
}

func testCtx(t *testing.T) context.Context {
	return cmdio.MockDiscard(t.Context())
}

func TestRenderDebug(t *testing.T) {
	input := fakeSSE(`{"type":"reasoning"}`, `{"type":"message"}`)
	var buf bytes.Buffer
	err := RenderDebug(strings.NewReader(input), &buf)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"reasoning"`)
	assert.Contains(t, lines[1], `"message"`)
}

func TestRenderText_MessageAndThinking(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"reasoning": {{Kind: EventThinking, Text: "Thinking..."}},
		"message":   {{Kind: EventText, Text: "Total sales were $1,234,567."}},
		"done":      {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("reasoning", "message", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "Total sales were $1,234,567.")
	assert.Empty(t, stderr.String())
}

func TestRenderText_SQLExecution(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql",
			Arguments: `{"sql":"SELECT SUM(amount) FROM sales","title":"Total Sales"}`,
		}}},
		"message": {{Kind: EventText, Text: "Sales summed."}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "message", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{ShowSQL: true})
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "SQL executed (Total Sales):")
	assert.Contains(t, stdout.String(), "SELECT SUM(amount) FROM sales")
}

func TestRenderText_SQLHiddenByDefault(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql",
			Arguments: `{"sql":"SELECT 1","title":"Test"}`,
		}}},
		"message": {{Kind: EventText, Text: "Done."}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "message", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.NotContains(t, stdout.String(), "SQL executed")
}

func TestRenderText_ExecuteSQLQuery(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql_query",
			Arguments: `{"query":"SELECT * FROM kie.test.bbc_articles LIMIT 3","thought":"Let me sample the table"}`,
		}}},
		"message": {{Kind: EventText, Text: "Sampled."}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "message", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{ShowSQL: true})
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "SQL executed:")
	assert.Contains(t, stdout.String(), "SELECT * FROM kie.test.bbc_articles LIMIT 3")
}

func TestRenderText_Error(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"error": {{Kind: EventError, Text: "No eligible SQL warehouse found", ErrorCode: "RESOURCE_DOES_NOT_EXIST"}},
	})
	input := fakeSSE("error")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESOURCE_DOES_NOT_EXIST")
	assert.Contains(t, err.Error(), "No eligible SQL warehouse found")
	// The root command prints returned errors; the renderer must not print
	// the error a second time.
	assert.Empty(t, stderr.String())
}

func TestRenderText_ErrorWithoutCode(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"error": {{Kind: EventError, Text: "boom"}},
	})
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(fakeSSE("error")), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Equal(t, "API error: boom", err.Error())
}

func TestRenderText_FailedResponseStatus(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"msg":  {{Kind: EventText, Text: "Some text"}},
		"done": {{Kind: EventDone, Status: "failed"}},
	})
	input := fakeSSE("msg", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestRenderText_ThoughtsNotInStdout(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"thought":  {{Kind: EventThinking, Text: "I will search for tables..."}},
		"response": {{Kind: EventText, Text: "Here are your tables."}},
		"done":     {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("thought", "response", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.NotContains(t, stdout.String(), "I will search for tables...")
	assert.Contains(t, stdout.String(), "Here are your tables.")
}

func TestRenderText_FinalResponseOnly(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"final": {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":  {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("final", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "The answer is 42.")
}

func TestRenderText_FinalResponseNotSuppressedByIntermediateText(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"preamble": {{Kind: EventText, Text: "Let me look at the sales table first."}},
		"final":    {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":     {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("preamble", "final", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Let me look at the sales table first.")
	assert.Contains(t, stdout.String(), "The answer is 42.")
}

func TestRenderText_FinalResponseDuplicateSuppressed(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "The answer is 42."}},
		"final":   {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("message", "final", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(stdout.String(), "The answer is 42."))
}

func TestRenderText_NoAnswerFails(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"thinking": {{Kind: EventThinking, Text: "Thinking..."}},
		"done":     {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("thinking", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without an answer")
	assert.Contains(t, err.Error(), "--raw")
}

func TestRenderText_MissingDoneWarns(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "Partial answer."}},
	})
	input := fakeSSE("message")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Partial answer.")
	assert.Contains(t, stderr.String(), "may be incomplete")
}

func TestRenderText_UnparsedEventsWarn(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "Answer."}},
		"junk":    {{Kind: EventUnparsed, Raw: "junk"}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("message", "junk", "junk", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "2 stream event(s) could not be parsed")
}

func TestRenderText_VizChartAfterText(t *testing.T) {
	viz := &VizEvent{
		Spec: &VizSpec{Title: "Totals", WidgetType: widgetBar, XField: "name", YFields: []string{"total"}},
		Data: &TableData{Columns: []string{"name", "total"}, Rows: [][]string{{"Alpha", "100"}}},
	}
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "Here is the breakdown."}},
		"viz":     {{Kind: EventViz, Viz: viz}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	// Viz arrives before the text: the chart must still render after it.
	input := fakeSSE("viz", "message", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	out := stdout.String()
	assert.Less(t, strings.Index(out, "Here is the breakdown."), strings.Index(out, "Totals"))
	assert.Contains(t, out, "█")
	assert.NotContains(t, out, "\033[", "color disabled must not emit ANSI escapes")
}

func TestRenderText_VizPlaceholderWhenUnrenderable(t *testing.T) {
	viz := &VizEvent{
		Spec: &VizSpec{Title: "Totals", WidgetType: widgetBar, XField: "name", YFields: []string{"missing"}},
		Data: &TableData{Columns: []string{"name", "total"}, Rows: [][]string{{"Alpha", "100"}}},
	}
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "See the chart below."}},
		"viz":     {{Kind: EventViz, Viz: viz}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("message", "viz", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(testCtx(t), strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `[chart "Totals" could not be rendered in the terminal]`)
}

func TestRenderJSON_FullStream(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"thinking": {{Kind: EventThinking, Text: "Thinking..."}},
		"text":     {{Kind: EventText, Text: "Total sales were $1,234,567."}},
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql",
			Arguments: `{"sql":"SELECT SUM(amount) FROM sales","title":"Total Sales"}`,
		}}},
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("thinking", "text", "sql", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "Total sales were $1,234,567.", result.Text)
	assert.Empty(t, result.Error)
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "execute_sql", result.ToolCalls[0].Name)
	assert.Equal(t, "SELECT SUM(amount) FROM sales", result.ToolCalls[0].SQL)
	assert.Equal(t, "Total Sales", result.ToolCalls[0].Title)
}

func TestRenderJSON_AppendsTextEvents(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"text": {
			{Kind: EventText, Text: "Total sales "},
			{Kind: EventText, Text: "were $1,234,567."},
		},
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("text", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "Total sales were $1,234,567.", result.Text)
}

func TestRenderJSON_ErrorEvent(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"error": {{Kind: EventError, Text: "No eligible SQL warehouse found", ErrorCode: "RESOURCE_DOES_NOT_EXIST"}},
	})
	input := fakeSSE("error")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESOURCE_DOES_NOT_EXIST")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "No eligible SQL warehouse")
}

func TestRenderJSON_ErrorPreservesAccumulatedText(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"text":  {{Kind: EventText, Text: "Partial answer."}},
		"error": {{Kind: EventError, Text: "backend exploded", ErrorCode: "INTERNAL"}},
	})
	input := fakeSSE("text", "error")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.Error(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "Partial answer.", result.Text)
	assert.Equal(t, "backend exploded", result.Error)
	assert.Equal(t, "error", result.Status)
}

func TestRenderJSON_FailedResponseStatus(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"msg":  {{Kind: EventText, Text: "Some text"}},
		"done": {{Kind: EventDone, Status: "failed"}},
	})
	input := fakeSSE("msg", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
}

func TestRenderJSON_EmptyStreamFails(t *testing.T) {
	// A stream that produced nothing must not report success: the historical
	// failure mode was a wire format change that dropped every item.
	adapt := testAdapter(map[string][]StreamEvent{})
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(""), &buf, &stderr, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without an answer")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
	assert.NotEmpty(t, result.Error)
}

func TestRenderJSON_MissingDoneIsIncomplete(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"text": {{Kind: EventText, Text: "Partial answer."}},
	})
	input := fakeSSE("text")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without a completion event")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "incomplete", result.Status)
	assert.Equal(t, "Partial answer.", result.Text)
}

func TestRenderJSON_FinalResponseFallback(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"final": {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":  {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("final", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "The answer is 42.", result.Text)
}

func TestRenderJSON_FinalResponseAppendedToDistinctText(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"preamble": {{Kind: EventText, Text: "Let me check."}},
		"final":    {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":     {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("preamble", "final", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "Let me check.\n\nThe answer is 42.", result.Text)
}

func TestRenderJSON_FinalResponseDuplicateSuppressed(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"message": {{Kind: EventText, Text: "The answer is 42."}},
		"final":   {{Kind: EventFinalResponse, Text: "The answer is 42."}},
		"done":    {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("message", "final", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "The answer is 42.", result.Text)
}

func TestRenderJSON_ReadErrorStillWritesJSON(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"text": {{Kind: EventText, Text: "Partial"}},
	})
	r := io.MultiReader(strings.NewReader(fakeSSE("text")), iotest.ErrReader(errors.New("connection reset")))
	var buf, stderr bytes.Buffer
	err := RenderJSON(r, &buf, &stderr, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "connection reset")
}

func TestRenderJSON_ExecuteSQLQuery(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql_query",
			Arguments: `{"query":"SELECT COUNT(*) FROM kie.test.bbc_articles","thought":"Check row count"}`,
		}}},
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "done")
	var buf, stderr bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, &stderr, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "execute_sql_query", result.ToolCalls[0].Name)
	assert.Equal(t, "SELECT COUNT(*) FROM kie.test.bbc_articles", result.ToolCalls[0].SQL)
}

func TestTruncateStatus(t *testing.T) {
	assert.Equal(t, "one line", truncateStatus("one\nline"))
	long := strings.Repeat("x", 2*maxStatusRunes)
	got := truncateStatus(long)
	assert.Len(t, []rune(got), maxStatusRunes)
	assert.True(t, strings.HasSuffix(got, "..."))
}
