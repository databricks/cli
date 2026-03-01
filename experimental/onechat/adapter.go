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

// AdaptSSE converts a raw OneChat SSE data payload into StreamEvents.
// It handles error events, response.done/completed, and response.output_item.added.
// The .updated and .done item variants are ignored to prevent duplicate rendering.
func AdaptSSE(data string) []agentstream.StreamEvent {
	var envelope SSEEventEnvelope
	if err := json.Unmarshal([]byte(data), &envelope); err != nil {
		return nil
	}

	switch envelope.Type {
	case eventError:
		return adaptError(data)
	case eventResponseDone, eventResponseComplete:
		return adaptResponseDone(data)
	case eventOutputItemAdded:
		return adaptOutputItem(data)
	default:
		// Ignore .updated, .done item variants and other event types.
		return nil
	}
}

func adaptError(data string) []agentstream.StreamEvent {
	var sseErr SSEError
	if err := json.Unmarshal([]byte(data), &sseErr); err != nil || sseErr.Message == "" {
		return nil
	}
	return []agentstream.StreamEvent{{
		Kind:      agentstream.EventError,
		Text:      sseErr.Message,
		ErrorCode: sseErr.ErrorCodeString(),
		Raw:       data,
	}}
}

func adaptResponseDone(data string) []agentstream.StreamEvent {
	var done SSEResponseDone
	if err := json.Unmarshal([]byte(data), &done); err != nil {
		return nil
	}
	return []agentstream.StreamEvent{{
		Kind:   agentstream.EventDone,
		Status: done.Response.Status,
		Raw:    data,
	}}
}

func adaptOutputItem(data string) []agentstream.StreamEvent {
	var event SSEOutputItemEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil
	}

	item := event.Item
	switch item.Type {
	case itemReasoning:
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventThinking,
			Text: "Thinking...",
			Raw:  data,
		}}
	case itemMessage:
		return adaptMessage(item, data)
	case itemFunctionCall:
		return []agentstream.StreamEvent{{
			Kind: agentstream.EventToolCall,
			ToolCall: &agentstream.ToolCallEvent{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
			Raw: data,
		}}
	default:
		return nil
	}
}

func adaptMessage(item OutputItem, data string) []agentstream.StreamEvent {
	if item.Role != "assistant" {
		return nil
	}

	if item.UIType() == uiTypeThought {
		// Thoughts become status-line updates.
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				return []agentstream.StreamEvent{{
					Kind: agentstream.EventThinking,
					Text: c.Text,
					Raw:  data,
				}}
			}
		}
		return nil
	}

	// FINAL_RESPONSE or unlabeled assistant messages become text output.
	var events []agentstream.StreamEvent
	for _, c := range item.Content {
		if c.Type == "output_text" && c.Text != "" {
			events = append(events, agentstream.StreamEvent{
				Kind: agentstream.EventText,
				Text: c.Text,
				Raw:  data,
			})
		}
	}
	return events
}
