package genie

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/databricks/cli/experimental/agentstream"
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

// AdaptSSE converts a raw Genie SSE data payload into StreamEvents.
// This is the stateless version for backward compatibility. It does not
// track query results, so it cannot emit viz events.
func AdaptSSE(data string) []agentstream.StreamEvent {
	return adaptStateless(data)
}

// NewAdaptSSE creates a stateful adapter that tracks SQL query results
// and emits EventViz events when visualizations are encountered.
func NewAdaptSSE() agentstream.AdapterFunc {
	queryData := make(map[string]*agentstream.TableData) // statement_id -> parsed table
	processed := make(map[string]bool)                   // item ID -> already handled

	return func(data string) []agentstream.StreamEvent {
		b := []byte(data)

		var envelope SSEEventEnvelope
		if err := json.Unmarshal(b, &envelope); err != nil {
			return nil
		}

		switch envelope.Type {
		case eventError:
			return adaptError(b, data)
		case eventResponseDone, eventResponseComplete:
			return adaptResponseDone(b, data)
		case eventOutputItemAdded, eventOutputItemDone:
			return adaptOutputItemStateful(b, data, queryData, processed)
		default:
			return nil
		}
	}
}

func adaptStateless(data string) []agentstream.StreamEvent {
	b := []byte(data)

	var envelope SSEEventEnvelope
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil
	}

	switch envelope.Type {
	case eventError:
		return adaptError(b, data)
	case eventResponseDone, eventResponseComplete:
		return adaptResponseDone(b, data)
	case eventOutputItemAdded:
		return adaptOutputItem(b, data)
	default:
		return nil
	}
}

func adaptError(b []byte, raw string) []agentstream.StreamEvent {
	var sseErr SSEError
	if err := json.Unmarshal(b, &sseErr); err != nil || sseErr.Message == "" {
		return nil
	}
	return []agentstream.StreamEvent{{
		Kind:      agentstream.EventError,
		Text:      sseErr.Message,
		ErrorCode: sseErr.ErrorCodeString(),
		Raw:       raw,
	}}
}

func adaptResponseDone(b []byte, raw string) []agentstream.StreamEvent {
	var done SSEResponseDone
	if err := json.Unmarshal(b, &done); err != nil {
		return nil
	}
	return []agentstream.StreamEvent{{
		Kind:   agentstream.EventDone,
		Status: done.Response.Status,
		Raw:    raw,
	}}
}

// adaptOutputItem handles output items without state (original behavior).
func adaptOutputItem(b []byte, raw string) []agentstream.StreamEvent {
	var event SSEOutputItemEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return nil
	}

	item := event.Item
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
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventToolCall,
			ToolCall: &agentstream.ToolCallEvent{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
			Raw: raw,
		}}
	default:
		return nil
	}
}

// itemTypeEnvelope extracts just the item type from an SSE event.
// This avoids parsing metadata (which may contain non-string values
// like booleans and objects that break map[string]string).
type itemTypeEnvelope struct {
	Item struct {
		Type string `json:"type"`
	} `json:"item"`
}

// adaptOutputItemStateful handles output items with query result tracking.
func adaptOutputItemStateful(b []byte, raw string, queryData map[string]*agentstream.TableData, processed map[string]bool) []agentstream.StreamEvent {
	// First pass: detect item type without parsing metadata.
	var typeCheck itemTypeEnvelope
	if err := json.Unmarshal(b, &typeCheck); err != nil {
		return nil
	}

	switch typeCheck.Item.Type {
	case itemFunctionCallOutput:
		// Parse with funcCallOutputEvent which handles non-string metadata.
		return adaptFuncCallOutput(b, raw, queryData, processed)
	default:
		// For other item types, use the original struct (metadata is string-only).
		return adaptOutputItem(b, raw)
	}
}

// adaptFuncCallOutput processes function_call_output items.
// It stores SQL query results and emits EventViz for visualizations.
func adaptFuncCallOutput(b []byte, raw string, queryData map[string]*agentstream.TableData, processed map[string]bool) []agentstream.StreamEvent {
	var event funcCallOutputEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return nil
	}

	item := event.Item
	if item.Status != "completed" {
		return nil
	}

	// Dedup: skip if we already processed this item.
	if item.ID != "" {
		if processed[item.ID] {
			return nil
		}
		processed[item.ID] = true
	}

	meta := item.Metadata

	// SQL query result: store the preview rows so a later viz can join to them.
	if meta.UIType == uiTypeQueryExec && meta.StatementID != "" {
		if td := tableDataFromResult(meta.ResultData); td != nil {
			queryData[meta.StatementID] = td
		}
		return nil
	}

	// Visualization: build a VizEvent from the viz_definition spec + stored data.
	if meta.UIType == uiTypeViz && meta.VizDefinition != "" {
		spec := vizSpecFromVizDefinition(meta.VizDefinition)
		if spec == nil {
			return nil
		}

		// Look up query data by sql_id (which matches the SQL output's statement_id).
		var data *agentstream.TableData
		if meta.SQLID != "" {
			data = queryData[meta.SQLID]
		}
		if data == nil && meta.StatementID != "" {
			data = queryData[meta.StatementID]
		}

		return []agentstream.StreamEvent{{
			Kind: agentstream.EventViz,
			Viz: &agentstream.VizEvent{
				Spec: spec,
				Data: data,
			},
			Raw: raw,
		}}
	}

	return nil
}

// tableDataFromResult converts QUERY_EXECUTION result_data (columns plus
// preview rows in array-of-arrays form) into TableData. Returns nil when there
// are no rows to render.
func tableDataFromResult(rd *queryResultData) *agentstream.TableData {
	if rd == nil || len(rd.PreviewRows) == 0 {
		return nil
	}
	columns := make([]string, len(rd.Columns))
	for i, c := range rd.Columns {
		columns[i] = c.Name
	}
	rows := make([][]string, 0, len(rd.PreviewRows))
	for _, raw := range rd.PreviewRows {
		row := make([]string, len(raw))
		for i, v := range raw {
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
