package onechat

import (
	"testing"

	"github.com/databricks/cli/experimental/agentstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Raw SSE data payloads (without "data: " prefix or trailing newlines).
const reasoningData = `{"type":"response.output_item.added","output_index":0,"item":{"type":"reasoning","id":"r1","status":"in_progress","content":[{"type":"reasoning_text","text":"Looking at data..."}]}}`

const messageData = `{"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Total sales were $1,234,567.","annotations":[]}]}}`

const thoughtData = `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"t1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"I will search for tables...","annotations":[]}],"metadata":{"ui_type":"THOUGHT"}}}`

const finalResponseData = `{"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Here are your tables.","annotations":[]}],"metadata":{"ui_type":"FINAL_RESPONSE"}}}`

const sqlData = `{"type":"response.output_item.added","output_index":2,"item":{"type":"function_call","id":"fc1","call_id":"c1","status":"completed","name":"execute_sql","arguments":"{\"sql\":\"SELECT SUM(amount) FROM sales\",\"title\":\"Total Sales\"}"}}`

const doneData = `{"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[]}}`

const doneFailedData = `{"type":"response.done","response":{"id":"resp_1","status":"failed","output":[]}}`

const errorData = `{"type":"error","error_code":"RESOURCE_DOES_NOT_EXIST","message":"No eligible SQL warehouse found"}`

const messageUpdatedData = `{"type":"response.output_item.updated","output_index":1,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Total sales were $1,234,567.","annotations":[]}]}}`

const messageDoneData = `{"type":"response.output_item.done","output_index":1,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Total sales were $1,234,567.","annotations":[]}]}}`

func TestAdaptSSE_Reasoning(t *testing.T) {
	events := AdaptSSE(reasoningData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventThinking, events[0].Kind)
	assert.Equal(t, "Thinking...", events[0].Text)
}

func TestAdaptSSE_Message(t *testing.T) {
	events := AdaptSSE(messageData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "Total sales were $1,234,567.", events[0].Text)
}

func TestAdaptSSE_Thought(t *testing.T) {
	events := AdaptSSE(thoughtData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventThinking, events[0].Kind)
	assert.Equal(t, "I will search for tables...", events[0].Text)
}

func TestAdaptSSE_FinalResponse(t *testing.T) {
	events := AdaptSSE(finalResponseData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "Here are your tables.", events[0].Text)
}

func TestAdaptSSE_FunctionCall(t *testing.T) {
	events := AdaptSSE(sqlData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventToolCall, events[0].Kind)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, "execute_sql", events[0].ToolCall.Name)
	assert.Contains(t, events[0].ToolCall.Arguments, "SELECT SUM(amount) FROM sales")
}

func TestAdaptSSE_Error(t *testing.T) {
	events := AdaptSSE(errorData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventError, events[0].Kind)
	assert.Equal(t, "No eligible SQL warehouse found", events[0].Text)
	assert.Equal(t, "RESOURCE_DOES_NOT_EXIST", events[0].ErrorCode)
}

func TestAdaptSSE_DoneCompleted(t *testing.T) {
	events := AdaptSSE(doneData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventDone, events[0].Kind)
	assert.Equal(t, "completed", events[0].Status)
}

func TestAdaptSSE_DoneFailed(t *testing.T) {
	events := AdaptSSE(doneFailedData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventDone, events[0].Kind)
	assert.Equal(t, "failed", events[0].Status)
}

func TestAdaptSSE_IgnoresUpdatedAndDoneItemVariants(t *testing.T) {
	assert.Empty(t, AdaptSSE(messageUpdatedData))
	assert.Empty(t, AdaptSSE(messageDoneData))
}

func TestAdaptSSE_InvalidJSON(t *testing.T) {
	assert.Empty(t, AdaptSSE("not json"))
}

func TestAdaptSSE_UnknownEventType(t *testing.T) {
	assert.Empty(t, AdaptSSE(`{"type":"response.unknown"}`))
}
