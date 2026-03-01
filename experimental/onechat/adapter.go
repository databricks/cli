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
)

// Output item type constants.
const (
	itemReasoning    = "reasoning"
	itemMessage      = "message"
	itemFunctionCall = "function_call"
)

// UI type metadata constants.
const (
	uiTypeThought       = "THOUGHT"
	uiTypeFinalResponse = "FINAL_RESPONSE"
)

// Content and role constants.
const (
	roleAssistant     = "assistant"
	contentOutputText = "output_text"
)

// AdaptSSE converts a raw OneChat SSE data payload into StreamEvents.
// It handles error events, response.done/completed, and response.output_item.added.
// The .updated and .done item variants are ignored to prevent duplicate rendering.
func AdaptSSE(data string) []agentstream.StreamEvent {
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
		// Ignore .updated, .done item variants and other event types.
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
