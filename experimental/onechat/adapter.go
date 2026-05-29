package onechat

import (
	"encoding/json"

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

// AdaptSSE converts a raw OneChat SSE data payload into StreamEvents.
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

	// SQL query result: parse and store the markdown table.
	if meta.UIType == uiTypeQueryExec && meta.StatementID != "" {
		td := agentstream.ParseMarkdownTable(item.Output)
		if td != nil {
			queryData[meta.StatementID] = td
		}
		return nil
	}

	// Visualization: build a VizEvent from the render_spec + stored query data.
	if meta.UIType == uiTypeViz && meta.RenderSpec != nil {
		spec := vizSpecFromRenderSpec(meta.RenderSpec)
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

// vizSpecFromRenderSpec converts a renderSpecJSON into a VizSpec.
func vizSpecFromRenderSpec(rs *renderSpecJSON) *agentstream.VizSpec {
	spec := &agentstream.VizSpec{
		Title:      rs.Frame.Title,
		WidgetType: rs.WidgetType,
		Layout:     rs.Mark.Layout,
	}
	if rs.Encodings.X != nil {
		spec.XField = rs.Encodings.X.FieldName
		spec.XTitle = rs.Encodings.X.Axis.Title
	}
	if rs.Encodings.Y != nil {
		spec.YFields = []string{rs.Encodings.Y.FieldName}
		spec.YTitle = rs.Encodings.Y.Axis.Title
	}
	if rs.Encodings.Color != nil {
		spec.ColorField = rs.Encodings.Color.FieldName
	}
	return spec
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
