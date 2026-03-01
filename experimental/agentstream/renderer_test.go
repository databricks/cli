package agentstream

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

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
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Waiting for response...")
	assert.Contains(t, stderr.String(), "Thinking...")
	assert.Contains(t, stdout.String(), "Total sales were $1,234,567.")
}

func TestRenderText_SQLExecution(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql",
			Arguments: `{"sql":"SELECT SUM(amount) FROM sales","title":"Total Sales"}`,
		}}},
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{ShowSQL: true})
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
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.NotContains(t, stdout.String(), "SQL executed")
}

func TestRenderText_ExecuteSQLQuery(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"sql": {{Kind: EventToolCall, ToolCall: &ToolCallEvent{
			Name:      "execute_sql_query",
			Arguments: `{"query":"SELECT * FROM kie.test.bbc_articles LIMIT 3","thought":"Let me sample the table"}`,
		}}},
		"done": {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("sql", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{ShowSQL: true})
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
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "No eligible SQL warehouse found")
}

func TestRenderText_FailedResponseStatus(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"msg":  {{Kind: EventText, Text: "Some text"}},
		"done": {{Kind: EventDone, Status: "failed"}},
	})
	input := fakeSSE("msg", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestRenderText_ThoughtsGoToStderr(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"thought":  {{Kind: EventThinking, Text: "I will search for tables..."}},
		"response": {{Kind: EventText, Text: "Here are your tables."}},
		"done":     {{Kind: EventDone, Status: "completed"}},
	})
	input := fakeSSE("thought", "response", "done")
	var stdout, stderr bytes.Buffer
	err := RenderText(strings.NewReader(input), &stdout, &stderr, adapt, RenderOptions{})
	require.NoError(t, err)

	assert.NotContains(t, stdout.String(), "I will search for tables...")
	assert.Contains(t, stderr.String(), "I will search for tables...")
	assert.Contains(t, stdout.String(), "Here are your tables.")
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
	var buf bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "Total sales were $1,234,567.", result.Text)
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "execute_sql", result.ToolCalls[0].Name)
	assert.Equal(t, "SELECT SUM(amount) FROM sales", result.ToolCalls[0].SQL)
	assert.Equal(t, "Total Sales", result.ToolCalls[0].Title)
}

func TestRenderJSON_ErrorEvent(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"error": {{Kind: EventError, Text: "No eligible SQL warehouse found", ErrorCode: "RESOURCE_DOES_NOT_EXIST"}},
	})
	input := fakeSSE("error")
	var buf bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESOURCE_DOES_NOT_EXIST")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Text, "No eligible SQL warehouse")
}

func TestRenderJSON_FailedResponseStatus(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{
		"msg":  {{Kind: EventText, Text: "Some text"}},
		"done": {{Kind: EventDone, Status: "failed"}},
	})
	input := fakeSSE("msg", "done")
	var buf bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, adapt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "error", result.Status)
}

func TestRenderJSON_EmptyStream(t *testing.T) {
	adapt := testAdapter(map[string][]StreamEvent{})
	var buf bytes.Buffer
	err := RenderJSON(strings.NewReader(""), &buf, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "completed", result.Status)
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
	var buf bytes.Buffer
	err := RenderJSON(strings.NewReader(input), &buf, adapt)
	require.NoError(t, err)

	var result StreamResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "execute_sql_query", result.ToolCalls[0].Name)
	assert.Equal(t, "SELECT COUNT(*) FROM kie.test.bbc_articles", result.ToolCalls[0].SQL)
}

func TestRenderMarkdown_StripsEmbeddedBlocks(t *testing.T) {
	input := "Some text\n<!-- begin-embedded:query_abc -->\n| A | B |\n| --- | --- |\n| 1 | 2 |\n<!-- end-embedded:query_abc -->\nMore text"
	var buf bytes.Buffer
	renderMarkdown(&buf, input)
	out := buf.String()

	assert.NotContains(t, out, "begin-embedded")
	assert.NotContains(t, out, "end-embedded")
	assert.Contains(t, out, "Some text")
	assert.Contains(t, out, "More text")
}
