package genie

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/experimental/genie/agentstream"
)

// SSE event type constants.
const (
	eventError            = "error"
	eventResponseDone     = "response.done"
	eventResponseComplete = "response.completed"
	eventOutputItemAdded  = "response.output_item.added"
	eventOutputItemDone   = "response.output_item.done"
)

// Output item type constants.
const (
	itemReasoning          = "reasoning"
	itemMessage            = "message"
	itemFunctionCall       = "function_call"
	itemFunctionCallOutput = "function_call_output"
)

// Item status constants. An in_progress item can carry partial content that
// its later .done event completes; an empty status is treated as complete
// because some items (e.g. function_call .added events) omit the field.
const (
	itemStatusCompleted  = "completed"
	itemStatusInProgress = "in_progress"
)

// UI type metadata constants.
const (
	uiTypeThought       = "THOUGHT"
	uiTypeFinalResponse = "FINAL_RESPONSE"
	uiTypeViz           = "VIZ"
	uiTypeQueryExec     = "QUERY_EXECUTION"
)

// Content and role constants.
const (
	roleAssistant     = "assistant"
	contentOutputText = "output_text"
)

// Tool names that carry user-facing output rather than hidden tool activity:
// output_final_response delivers the answer, ask_user_questions asks the user
// to clarify. Other tool calls are intermediate activity and stay hidden.
const (
	toolOutputFinalResponse = "output_final_response"
	toolAskUserQuestions    = "ask_user_questions"
)

// sseAdapter tracks per-stream state: parsed SQL results, which item IDs have
// already been handled (.added and .done can both arrive for one item), and
// visualizations waiting for their result data.
type sseAdapter struct {
	queryData  map[string]*agentstream.TableData // statement_id -> parsed table
	processed  map[string]bool                   // item ID -> already handled
	pendingViz []pendingViz                      // viz specs seen before their data
}

// pendingViz is a visualization whose QUERY_EXECUTION data has not arrived
// yet, keyed by the statement IDs it joins on.
type pendingViz struct {
	spec   *agentstream.VizSpec
	sqlID  string
	stmtID string
	raw    string
}

// NewAdaptSSE creates a stateful adapter that tracks SQL query results
// and emits EventViz events when visualizations are encountered.
//
// Unknown event types are ignored (the server may add new ones), but a known
// event whose payload fails to decode becomes EventUnparsed: dropping those
// silently is how a metadata type change once made this command print
// nothing and exit 0.
func NewAdaptSSE() agentstream.AdapterFunc {
	a := &sseAdapter{
		queryData: make(map[string]*agentstream.TableData),
		processed: make(map[string]bool),
	}
	return a.adapt
}

func (a *sseAdapter) adapt(data string) []agentstream.StreamEvent {
	b := []byte(data)

	var envelope SSEEventEnvelope
	if err := json.Unmarshal(b, &envelope); err != nil {
		return unparsed(data)
	}

	switch envelope.Type {
	case eventError:
		return adaptError(b, data)
	case eventResponseDone, eventResponseComplete:
		return adaptResponseDone(b, data)
	case eventOutputItemAdded, eventOutputItemDone:
		return a.adaptOutputItem(b, data)
	default:
		return nil
	}
}

// unparsed flags a payload the adapter recognized but could not decode.
func unparsed(raw string) []agentstream.StreamEvent {
	return []agentstream.StreamEvent{{
		Kind: agentstream.EventUnparsed,
		Raw:  raw,
	}}
}

// adaptError converts a server error event. It never returns nil: an error
// event in an unexpected shape must still fail the stream loudly, so the raw
// payload stands in when no message can be extracted.
func adaptError(b []byte, raw string) []agentstream.StreamEvent {
	var sseErr SSEError
	_ = json.Unmarshal(b, &sseErr)
	text := sseErr.Message
	if text == "" {
		text = raw
	}
	return []agentstream.StreamEvent{{
		Kind:      agentstream.EventError,
		Text:      text,
		ErrorCode: sseErr.ErrorCodeString(),
		Raw:       raw,
	}}
}

func adaptResponseDone(b []byte, raw string) []agentstream.StreamEvent {
	var done SSEResponseDone
	if err := json.Unmarshal(b, &done); err != nil {
		// An undecodable completion event must not pass for a successful
		// one; the renderers treat a stream without EventDone as incomplete.
		return unparsed(raw)
	}
	return []agentstream.StreamEvent{{
		Kind:           agentstream.EventDone,
		Status:         done.Response.Status,
		ConversationID: done.Response.ConversationID,
		Raw:            raw,
	}}
}

func adaptOutputItemValue(item OutputItem, raw string) []agentstream.StreamEvent {
	switch item.Type {
	case itemReasoning:
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventThinking,
			Text: "Thinking...",
			Raw:  raw,
		}}
	case itemMessage:
		return adaptMessage(item, raw)
	case itemFunctionCall:
		return adaptFunctionCall(item, raw)
	default:
		return nil
	}
}

// adaptFunctionCall converts a function_call item into stream events. Most tool
// calls become EventToolCall (hidden unless they are SQL with --include-sql),
// but two carry user-facing output that would otherwise be lost: the final
// answer (output_final_response) and clarification prompts (ask_user_questions).
func adaptFunctionCall(item OutputItem, raw string) []agentstream.StreamEvent {
	// An in_progress item can carry partial arguments that its .done event
	// completes; emitting it would mark the item processed and dedupe away
	// the complete version.
	if item.Status == itemStatusInProgress {
		return nil
	}
	switch item.Name {
	case toolOutputFinalResponse:
		text := finalResponseText(item.Arguments)
		if text == "" {
			return nil
		}
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventFinalResponse,
			Text: text,
			Raw:  raw,
		}}
	case toolAskUserQuestions:
		text := formatQuestions(item.Arguments)
		if text == "" {
			return nil
		}
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventText,
			Text: text,
			Raw:  raw,
		}}
	default:
		// The .added event for a tool call can carry empty arguments that are
		// only filled in on the matching .done event. Emitting the empty call
		// would mark the item as processed and dedupe away the real one.
		if item.Arguments == "" {
			return nil
		}
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventToolCall,
			ToolCall: &agentstream.ToolCallEvent{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
			Raw: raw,
		}}
	}
}

// finalResponseText extracts the answer from an output_final_response tool
// call's arguments ({"response": "..."}).
func finalResponseText(arguments string) string {
	var args struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}
	return args.Response
}

// formatQuestions renders an ask_user_questions tool call's arguments into a
// readable clarification prompt. Arguments are
// {"questions": [{"question", "type", "choices": [{"label", "description"}]}]}.
func formatQuestions(arguments string) string {
	var args struct {
		Questions []struct {
			Question string `json:"question"`
			Type     string `json:"type"`
			Choices  []struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"choices"`
		} `json:"questions"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}
	if len(args.Questions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("The agent needs clarification:\n")
	for i, q := range args.Questions {
		fmt.Fprintf(&b, "\n%d. %s", i+1, q.Question)
		if q.Type == "confirmation" {
			b.WriteString(" (yes / no)")
		}
		b.WriteString("\n")
		for _, c := range q.Choices {
			if c.Description != "" {
				fmt.Fprintf(&b, "   - %s: %s\n", c.Label, c.Description)
			} else {
				fmt.Fprintf(&b, "   - %s\n", c.Label)
			}
		}
	}
	b.WriteString("\nReply in plain text to continue.")
	return b.String()
}

// itemTypeEnvelope extracts just the item type from an SSE event.
// This avoids parsing metadata (which may contain non-string values
// like booleans and objects that break map[string]string).
type itemTypeEnvelope struct {
	Item struct {
		Type string `json:"type"`
	} `json:"item"`
}

// adaptOutputItem handles output items with query result tracking.
func (a *sseAdapter) adaptOutputItem(b []byte, raw string) []agentstream.StreamEvent {
	// First pass: detect item type without parsing metadata.
	var typeCheck itemTypeEnvelope
	if err := json.Unmarshal(b, &typeCheck); err != nil {
		return unparsed(raw)
	}

	switch typeCheck.Item.Type {
	case itemFunctionCallOutput:
		// Parse with funcCallOutputEvent which handles non-string metadata.
		return a.adaptFuncCallOutput(b, raw)
	default:
		var event SSEOutputItemEvent
		if err := json.Unmarshal(b, &event); err != nil {
			return unparsed(raw)
		}
		if event.Item.ID != "" {
			if a.processed[event.Item.ID] {
				return nil
			}
		}
		events := adaptOutputItemValue(event.Item, raw)
		if len(events) > 0 && event.Item.ID != "" {
			a.processed[event.Item.ID] = true
		}
		return events
	}
}

// adaptFuncCallOutput processes function_call_output items.
// It stores SQL query results and emits EventViz for visualizations.
func (a *sseAdapter) adaptFuncCallOutput(b []byte, raw string) []agentstream.StreamEvent {
	var event funcCallOutputEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return unparsed(raw)
	}

	item := event.Item
	if item.Status != itemStatusCompleted {
		return nil
	}

	if item.ID != "" && a.processed[item.ID] {
		return nil
	}

	meta := item.Metadata

	// SQL query result: store the preview rows so a viz can join to them.
	if meta.UIType == uiTypeQueryExec && meta.StatementID != "" {
		td := tableDataFromResult(meta.ResultData)
		if td == nil {
			return nil
		}
		a.queryData[meta.StatementID] = td
		if item.ID != "" {
			a.processed[item.ID] = true
		}
		return a.flushPendingViz(meta.StatementID)
	}

	// Visualization: build a VizEvent from the viz_definition spec + stored data.
	if meta.UIType == uiTypeViz && meta.VizDefinition != "" {
		spec := vizSpecFromVizDefinition(meta.VizDefinition)
		if spec == nil {
			return nil
		}

		if item.ID != "" {
			a.processed[item.ID] = true
		}

		// Look up query data by sql_id (which matches the SQL output's statement_id).
		if data := a.lookupQueryData(meta.SQLID, meta.StatementID); data != nil {
			return []agentstream.StreamEvent{vizEvent(spec, data, raw)}
		}

		// Items normally complete in causal order (the query before the viz
		// built from it), but a viz observed first is buffered rather than
		// silently lost.
		a.pendingViz = append(a.pendingViz, pendingViz{spec: spec, sqlID: meta.SQLID, stmtID: meta.StatementID, raw: raw})
		return nil
	}

	return nil
}

// lookupQueryData resolves the result table a viz joins to. sql_id is the
// primary key; statement_id appears on some payload variants.
func (a *sseAdapter) lookupQueryData(sqlID, stmtID string) *agentstream.TableData {
	if sqlID != "" {
		if td := a.queryData[sqlID]; td != nil {
			return td
		}
	}
	if stmtID != "" {
		return a.queryData[stmtID]
	}
	return nil
}

// flushPendingViz emits buffered visualizations whose data just arrived.
func (a *sseAdapter) flushPendingViz(stmtID string) []agentstream.StreamEvent {
	var events []agentstream.StreamEvent
	var remaining []pendingViz
	for _, p := range a.pendingViz {
		if p.sqlID == stmtID || p.stmtID == stmtID {
			events = append(events, vizEvent(p.spec, a.queryData[stmtID], p.raw))
			continue
		}
		remaining = append(remaining, p)
	}
	a.pendingViz = remaining
	return events
}

func vizEvent(spec *agentstream.VizSpec, data *agentstream.TableData, raw string) agentstream.StreamEvent {
	return agentstream.StreamEvent{
		Kind: agentstream.EventViz,
		Viz:  &agentstream.VizEvent{Spec: spec, Data: data},
		Raw:  raw,
	}
}

// tableDataFromResult converts QUERY_EXECUTION result_data (columns plus
// preview rows in array-of-arrays form) into TableData. Returns nil when there
// are no rows to render.
func tableDataFromResult(rd *queryResultData) *agentstream.TableData {
	if rd == nil || len(rd.Columns) == 0 || len(rd.PreviewRows) == 0 {
		return nil
	}
	columns := make([]string, len(rd.Columns))
	for i, c := range rd.Columns {
		columns[i] = c.Name
	}
	rows := make([][]string, 0, len(rd.PreviewRows))
	for _, raw := range rd.PreviewRows {
		row := make([]string, len(columns))
		for i, v := range raw[:min(len(raw), len(columns))] {
			row[i] = stringifyCell(v)
		}
		rows = append(rows, row)
	}
	return &agentstream.TableData{Columns: columns, Rows: rows}
}

// stringifyCell renders a preview-row cell as a string. The Statement Execution
// API returns cells as strings, but a numeric cell can arrive as a JSON number.
func stringifyCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// vizSpecFromVizDefinition parses the JSON-encoded Helios spec from
// metadata.viz_definition and maps it to a VizSpec. Returns nil if the spec
// can't be parsed or lacks usable x/y encodings.
func vizSpecFromVizDefinition(vizDefinition string) *agentstream.VizSpec {
	var def heliosVizDefinition
	if err := json.Unmarshal([]byte(vizDefinition), &def); err != nil {
		return nil
	}
	rs := def.RenderSpec
	if rs == nil {
		return nil
	}
	spec := &agentstream.VizSpec{
		Title:      rs.Frame.Title,
		WidgetType: rs.WidgetType,
	}
	if rs.Encodings.X != nil {
		spec.XField = encodingField(rs.Encodings.X)
		spec.XTitle = rs.Encodings.X.DisplayName
	}
	if rs.Encodings.Y != nil {
		spec.YFields = encodingFields(rs.Encodings.Y)
		spec.YTitle = rs.Encodings.Y.DisplayName
	}
	if spec.XField == "" || len(spec.YFields) == 0 {
		return nil
	}
	return spec
}

// encodingField returns the single field name for an axis: FieldName, or the
// first entry of Fields.
func encodingField(e *heliosEncoding) string {
	if e.FieldName != "" {
		return e.FieldName
	}
	if len(e.Fields) > 0 {
		return e.Fields[0].FieldName
	}
	return ""
}

// encodingFields returns every field name for an axis. A single FieldName
// yields one series; Fields yields one per entry (multi-series).
func encodingFields(e *heliosEncoding) []string {
	if len(e.Fields) > 0 {
		names := make([]string, 0, len(e.Fields))
		for _, f := range e.Fields {
			if f.FieldName != "" {
				names = append(names, f.FieldName)
			}
		}
		return names
	}
	if e.FieldName != "" {
		return []string{e.FieldName}
	}
	return nil
}

func adaptMessage(item OutputItem, raw string) []agentstream.StreamEvent {
	if item.Role != roleAssistant {
		return nil
	}

	// An in_progress message can carry partial text that its .done event
	// completes; emitting it would mark the item processed and dedupe away
	// the complete version.
	if item.Status == itemStatusInProgress {
		return nil
	}

	if item.UIType() == uiTypeThought {
		// Thoughts become status-line updates.
		for _, c := range item.Content {
			if c.Type == contentOutputText && c.Text != "" {
				return []agentstream.StreamEvent{{
					Kind: agentstream.EventThinking,
					Text: c.Text,
					Raw:  raw,
				}}
			}
		}
		return nil
	}

	// FINAL_RESPONSE or unlabeled assistant messages become text output.
	var events []agentstream.StreamEvent
	for _, c := range item.Content {
		if c.Type == contentOutputText && c.Text != "" {
			events = append(events, agentstream.StreamEvent{
				Kind: agentstream.EventText,
				Text: c.Text,
				Raw:  raw,
			})
		}
	}
	return events
}
