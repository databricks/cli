package genie

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

const sqlQueryData = `{"type":"response.output_item.added","output_index":9,"item":{"type":"function_call","id":"toolu_1","call_id":"toolu_1","name":"execute_sql_query","arguments":"{\"thought\":\"Let me sample the table\",\"query\":\"SELECT * FROM kie.test.bbc_articles LIMIT 3\"}","metadata":{"ui_type":"QUERY_EXECUTION","query":"SELECT * FROM kie.test.bbc_articles LIMIT 3"}}}`

const doneData = `{"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[]}}`

const doneFailedData = `{"type":"response.done","response":{"id":"resp_1","status":"failed","output":[]}}`

const errorData = `{"type":"error","error_code":"RESOURCE_DOES_NOT_EXIST","message":"No eligible SQL warehouse found"}`

const messageUpdatedData = `{"type":"response.output_item.updated","output_index":1,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Total sales were $1,234,567.","annotations":[]}]}}`

// adaptOne runs a single payload through a fresh stateful adapter.
func adaptOne(data string) []agentstream.StreamEvent {
	return NewAdaptSSE()(data)
}

func TestAdaptSSE_Reasoning(t *testing.T) {
	events := adaptOne(reasoningData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventThinking, events[0].Kind)
	assert.Equal(t, "Thinking...", events[0].Text)
}

func TestAdaptSSE_Message(t *testing.T) {
	events := adaptOne(messageData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "Total sales were $1,234,567.", events[0].Text)
}

func TestAdaptSSE_Thought(t *testing.T) {
	events := adaptOne(thoughtData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventThinking, events[0].Kind)
	assert.Equal(t, "I will search for tables...", events[0].Text)
}

func TestAdaptSSE_FinalResponse(t *testing.T) {
	events := adaptOne(finalResponseData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "Here are your tables.", events[0].Text)
}

func TestAdaptSSE_FunctionCall(t *testing.T) {
	events := adaptOne(sqlData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventToolCall, events[0].Kind)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, "execute_sql", events[0].ToolCall.Name)
	assert.Contains(t, events[0].ToolCall.Arguments, "SELECT SUM(amount) FROM sales")
}

func TestAdaptSSE_Error(t *testing.T) {
	events := adaptOne(errorData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventError, events[0].Kind)
	assert.Equal(t, "No eligible SQL warehouse found", events[0].Text)
	assert.Equal(t, "RESOURCE_DOES_NOT_EXIST", events[0].ErrorCode)
}

func TestAdaptSSE_ErrorInUnexpectedShape(t *testing.T) {
	// A server failure must never become an empty success, even when the
	// error payload doesn't match the expected shape.
	data := `{"type":"error","message":{"nested":"boom"}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventError, events[0].Kind)
	assert.Equal(t, data, events[0].Text)
	assert.Empty(t, events[0].ErrorCode)
}

func TestAdaptSSE_DoneCompleted(t *testing.T) {
	events := adaptOne(doneData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventDone, events[0].Kind)
	assert.Equal(t, "completed", events[0].Status)
}

func TestAdaptSSE_DoneFailed(t *testing.T) {
	events := adaptOne(doneFailedData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventDone, events[0].Kind)
	assert.Equal(t, "failed", events[0].Status)
}

func TestAdaptSSE_DoneInUnexpectedShape(t *testing.T) {
	// An undecodable completion event must not pass for a successful one.
	events := adaptOne(`{"type":"response.done","response":[1,2]}`)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventUnparsed, events[0].Kind)
}

func TestAdaptSSE_ExecuteSQLQuery(t *testing.T) {
	events := adaptOne(sqlQueryData)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventToolCall, events[0].Kind)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, "execute_sql_query", events[0].ToolCall.Name)
	assert.Contains(t, events[0].ToolCall.Arguments, "SELECT * FROM kie.test.bbc_articles LIMIT 3")
}

func TestAdaptSSE_UpdatedItemVariantIgnored(t *testing.T) {
	// Only .added and .done item events are processed; .updated duplicates
	// what .done will carry.
	assert.Empty(t, adaptOne(messageUpdatedData))
}

func TestAdaptSSE_InvalidJSON(t *testing.T) {
	events := adaptOne("not json")
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventUnparsed, events[0].Kind)
}

func TestAdaptSSE_UnknownEventTypeIgnored(t *testing.T) {
	assert.Empty(t, adaptOne(`{"type":"response.unknown"}`))
}

func TestAdaptSSE_UnparseableItemFlagged(t *testing.T) {
	// A known event type whose payload doesn't decode is exactly the failure
	// mode that once made the command print nothing: it must be flagged.
	data := `{"type":"response.output_item.added","item":{"type":"message","content":"not an array"}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventUnparsed, events[0].Kind)
}

func TestAdaptSSE_QueryExecutionAndViz(t *testing.T) {
	adapt := NewAdaptSSE()

	// A QUERY_EXECUTION output carries the preview rows under result_data.
	qe := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o1","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt1","result_data":{"columns":[{"name":"franchise"},{"name":"total"}],"preview_rows":[["Alpha","100"],["Beta","200"]]}}}}`
	assert.Empty(t, adapt(qe))

	// A VIZ output carries the Helios spec under viz_definition (a JSON string,
	// nested under renderSpec) and references the query by sql_id.
	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o2","call_id":"c2","status":"completed","metadata":{"ui_type":"VIZ","sql_id":"stmt1","embed_id":"v1","viz_definition":"{\"renderSpec\":{\"widgetType\":\"bar\",\"frame\":{\"title\":\"Total by Franchise\"},\"encodings\":{\"x\":{\"fieldName\":\"franchise\"},\"y\":{\"fieldName\":\"total\",\"displayName\":\"Total Sales\"}}}}"}}}`
	events := adapt(viz)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventViz, events[0].Kind)
	require.NotNil(t, events[0].Viz)
	require.NotNil(t, events[0].Viz.Spec)
	assert.Equal(t, "bar", events[0].Viz.Spec.WidgetType)
	assert.Equal(t, "Total by Franchise", events[0].Viz.Spec.Title)
	assert.Equal(t, "franchise", events[0].Viz.Spec.XField)
	assert.Equal(t, []string{"total"}, events[0].Viz.Spec.YFields)
	assert.Equal(t, "Total Sales", events[0].Viz.Spec.YTitle)
	require.NotNil(t, events[0].Viz.Data)
	assert.Equal(t, []string{"franchise", "total"}, events[0].Viz.Data.Columns)
	require.Len(t, events[0].Viz.Data.Rows, 2)
	assert.Equal(t, []string{"Beta", "200"}, events[0].Viz.Data.Rows[1])
}

func TestAdaptSSE_QueryExecutionCellTypes(t *testing.T) {
	adapt := NewAdaptSSE()

	// Preview cells can be JSON numbers, booleans, or null, and rows can be
	// shorter or longer than the column list.
	qe := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o1","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt1","result_data":{"columns":[{"name":"name"},{"name":"total"}],"preview_rows":[["Alpha",100.5],["Beta",true],["Gamma",null],["Delta"],["Epsilon","1","extra"]]}}}}`
	assert.Empty(t, adapt(qe))

	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o2","call_id":"c2","status":"completed","metadata":{"ui_type":"VIZ","sql_id":"stmt1","embed_id":"v1","viz_definition":"{\"renderSpec\":{\"widgetType\":\"bar\",\"frame\":{\"title\":\"T\"},\"encodings\":{\"x\":{\"fieldName\":\"name\"},\"y\":{\"fieldName\":\"total\"}}}}"}}}`
	events := adapt(viz)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].Viz.Data)
	assert.Equal(t, [][]string{
		{"Alpha", "100.5"},
		{"Beta", "true"},
		{"Gamma", ""},
		{"Delta", ""},
		{"Epsilon", "1"},
	}, events[0].Viz.Data.Rows)
}

func TestAdaptSSE_MultiSeriesYEncoding(t *testing.T) {
	adapt := NewAdaptSSE()

	qe := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o1","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt1","result_data":{"columns":[{"name":"month"},{"name":"a"},{"name":"b"}],"preview_rows":[["Jan","1","2"]]}}}}`
	assert.Empty(t, adapt(qe))

	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o2","call_id":"c2","status":"completed","metadata":{"ui_type":"VIZ","sql_id":"stmt1","embed_id":"v1","viz_definition":"{\"renderSpec\":{\"widgetType\":\"line\",\"frame\":{\"title\":\"T\"},\"encodings\":{\"x\":{\"fieldName\":\"month\"},\"y\":{\"fields\":[{\"fieldName\":\"a\"},{\"fieldName\":\"b\"}]}}}}"}}}`
	events := adapt(viz)
	require.Len(t, events, 1)
	assert.Equal(t, []string{"a", "b"}, events[0].Viz.Spec.YFields)
}

func TestAdaptSSE_QueryExecutionCanArriveAfterIncompleteOutput(t *testing.T) {
	adapt := NewAdaptSSE()

	incomplete := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o4","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt2"}}}`
	assert.Empty(t, adapt(incomplete))

	complete := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o4","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt2","result_data":{"columns":[{"name":"name"},{"name":"total"}],"preview_rows":[["Alpha","100"]]}}}}`
	assert.Empty(t, adapt(complete))

	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o5","call_id":"c2","status":"completed","metadata":{"ui_type":"VIZ","sql_id":"stmt2","embed_id":"v1","viz_definition":"{\"renderSpec\":{\"widgetType\":\"bar\",\"frame\":{\"title\":\"Total\"},\"encodings\":{\"x\":{\"fieldName\":\"name\"},\"y\":{\"fieldName\":\"total\"}}}}"}}}`
	events := adapt(viz)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].Viz.Data)
	assert.Equal(t, []string{"Alpha", "100"}, events[0].Viz.Data.Rows[0])
}

func TestNewAdaptSSE_VizBeforeQueryData(t *testing.T) {
	adapt := NewAdaptSSE()

	// The viz arrives before the query data it joins to: it must be buffered
	// and emitted when the data shows up, not silently lost.
	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o2","call_id":"c2","status":"completed","metadata":{"ui_type":"VIZ","sql_id":"stmt1","embed_id":"v1","viz_definition":"{\"renderSpec\":{\"widgetType\":\"bar\",\"frame\":{\"title\":\"T\"},\"encodings\":{\"x\":{\"fieldName\":\"name\"},\"y\":{\"fieldName\":\"total\"}}}}"}}}`
	assert.Empty(t, adapt(viz))

	qe := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o1","call_id":"c1","status":"completed","metadata":{"ui_type":"QUERY_EXECUTION","statement_id":"stmt1","result_data":{"columns":[{"name":"name"},{"name":"total"}],"preview_rows":[["Alpha","100"]]}}}}`
	events := adapt(qe)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventViz, events[0].Kind)
	require.NotNil(t, events[0].Viz.Data)
	assert.Equal(t, []string{"Alpha", "100"}, events[0].Viz.Data.Rows[0])
}

func TestNewAdaptSSE_InProgressMessageNotDedupedPrematurely(t *testing.T) {
	adapt := NewAdaptSSE()

	// A partial in_progress message must not be emitted (and must not mark
	// the item processed), or the complete .done version would be deduped.
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"m9","role":"assistant","status":"in_progress","content":[{"type":"output_text","text":"Partial an","annotations":[]}]}}`
	assert.Empty(t, adapt(added))

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"m9","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Partial answer completed.","annotations":[]}]}}`
	events := adapt(done)
	require.Len(t, events, 1)
	assert.Equal(t, "Partial answer completed.", events[0].Text)
}

func TestNewAdaptSSE_InProgressFunctionCallNotDedupedPrematurely(t *testing.T) {
	adapt := NewAdaptSSE()

	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f9","call_id":"c9","status":"in_progress","name":"execute_sql","arguments":"{\"sql\":\"SELECT"}}`
	assert.Empty(t, adapt(added))

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"f9","call_id":"c9","status":"completed","name":"execute_sql","arguments":"{\"sql\":\"SELECT 1\",\"title\":\"One\"}"}}`
	events := adapt(done)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventToolCall, events[0].Kind)
	assert.Contains(t, events[0].ToolCall.Arguments, "SELECT 1")
}

func TestAdaptSSE_VizWithoutDefinitionIgnored(t *testing.T) {
	adapt := NewAdaptSSE()
	viz := `{"type":"response.output_item.done","item":{"type":"function_call_output","id":"o3","call_id":"c3","status":"completed","metadata":{"ui_type":"VIZ","embed_id":"v2"}}}`
	assert.Empty(t, adapt(viz))
}

func TestAdaptSSE_OutputFinalResponse(t *testing.T) {
	data := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f1","call_id":"c1","name":"output_final_response","arguments":"{\"response\":\"The answer is 42.\"}"}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventFinalResponse, events[0].Kind)
	assert.Equal(t, "The answer is 42.", events[0].Text)
}

func TestAdaptSSE_AskUserQuestions(t *testing.T) {
	data := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f2","call_id":"c2","name":"ask_user_questions","arguments":"{\"questions\":[{\"question\":\"Which region?\",\"type\":\"choice\",\"choices\":[{\"label\":\"US\",\"description\":\"United States\"},{\"label\":\"EU\"}]}]}"}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Contains(t, events[0].Text, "Which region?")
	assert.Contains(t, events[0].Text, "US: United States")
	assert.Contains(t, events[0].Text, "EU")
}

func TestAdaptSSE_AskUserConfirmation(t *testing.T) {
	data := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f2","call_id":"c2","name":"ask_user_questions","arguments":"{\"questions\":[{\"question\":\"Proceed with all regions?\",\"type\":\"confirmation\"}]}"}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Text, "Proceed with all regions? (yes / no)")
}

func TestNewAdaptSSE_DedupsFunctionCall(t *testing.T) {
	adapt := NewAdaptSSE()
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f3","call_id":"c3","name":"output_final_response","arguments":"{\"response\":\"Hello.\"}"}}`
	events := adapt(added)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventFinalResponse, events[0].Kind)

	// The same item re-observed (e.g. on .done) must not emit a second time.
	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"f3","call_id":"c3","name":"output_final_response","arguments":"{\"response\":\"Hello.\"}"}}`
	assert.Empty(t, adapt(done))
}

func TestNewAdaptSSE_DoesNotDedupEmptyFunctionCall(t *testing.T) {
	adapt := NewAdaptSSE()
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f4","call_id":"c4","name":"output_final_response","arguments":""}}`
	assert.Empty(t, adapt(added))

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"f4","call_id":"c4","name":"output_final_response","arguments":"{\"response\":\"Hello after done.\"}"}}`
	events := adapt(done)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventFinalResponse, events[0].Kind)
	assert.Equal(t, "Hello after done.", events[0].Text)
}

func TestNewAdaptSSE_DoesNotDedupEmptyGenericToolCall(t *testing.T) {
	// The .added event for execute_sql can carry empty arguments that are
	// filled in on .done. Emitting the empty call would dedupe away the SQL.
	adapt := NewAdaptSSE()
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"f5","call_id":"c5","name":"execute_sql","arguments":""}}`
	assert.Empty(t, adapt(added))

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"f5","call_id":"c5","name":"execute_sql","arguments":"{\"sql\":\"SELECT 1\",\"title\":\"One\"}"}}`
	events := adapt(done)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventToolCall, events[0].Kind)
	require.NotNil(t, events[0].ToolCall)
	assert.Contains(t, events[0].ToolCall.Arguments, "SELECT 1")
}

func TestNewAdaptSSE_DedupsMessageAfterEmitting(t *testing.T) {
	adapt := NewAdaptSSE()
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"m2","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello.","annotations":[]}]}}`
	events := adapt(added)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"m2","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello.","annotations":[]}]}}`
	assert.Empty(t, adapt(done))
}

func TestNewAdaptSSE_MessageCanArriveOnDone(t *testing.T) {
	adapt := NewAdaptSSE()
	added := `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"m3","role":"assistant","status":"in_progress","content":[]}}`
	assert.Empty(t, adapt(added))

	done := `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"m3","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Done text.","annotations":[]}]}}`
	events := adapt(done)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "Done text.", events[0].Text)
}

func TestAdaptSSE_FinalResponseWithArrayMetadata(t *testing.T) {
	// Real backend metadata carries non-string values such as the
	// source_internal_ids array. The item must still parse and render; a
	// map[string]string metadata type silently dropped the whole message.
	data := `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"The answer.","annotations":[]}],"metadata":{"ui_type":"FINAL_RESPONSE","source_internal_ids":["msg_abc"],"response_id":"resp_1"}}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventText, events[0].Kind)
	assert.Equal(t, "The answer.", events[0].Text)
}

func TestAdaptSSE_ThoughtWithArrayMetadata(t *testing.T) {
	data := `{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"t1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Looking...","annotations":[]}],"metadata":{"ui_type":"THOUGHT","source_internal_ids":["msg_x"]}}}`
	events := adaptOne(data)
	require.Len(t, events, 1)
	assert.Equal(t, agentstream.EventThinking, events[0].Kind)
	assert.Equal(t, "Looking...", events[0].Text)
}
