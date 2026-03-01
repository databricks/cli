package onechat

// OneChatRequest is the request body for POST /api/2.0/data-rooms/tools/onechat/responses.
type OneChatRequest struct {
	Input       []InputItem `json:"input"`
	WarehouseID string      `json:"warehouseId,omitempty"`
}

// InputItem is a message in the input array.
type InputItem struct {
	Type    string        `json:"type"`
	Role    string        `json:"role,omitempty"`
	Content []ContentItem `json:"content,omitempty"`
}

// ContentItem is a content block within a message.
type ContentItem struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	Annotations []any  `json:"annotations,omitempty"`
}

// OutputItem represents an item in the SSE response stream.
type OutputItem struct {
	Type      string            `json:"type"`
	ID        string            `json:"id,omitempty"`
	Role      string            `json:"role,omitempty"`
	Status    string            `json:"status,omitempty"`
	Name      string            `json:"name,omitempty"`
	CallID    string            `json:"call_id,omitempty"`
	Arguments string            `json:"arguments,omitempty"`
	Content   []ContentItem     `json:"content,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// UIType returns the metadata ui_type field (e.g. "THOUGHT", "FINAL_RESPONSE").
func (o OutputItem) UIType() string {
	return o.Metadata["ui_type"]
}

// SSEOutputItemEvent is the payload for response.output_item.added events.
type SSEOutputItemEvent struct {
	Type        string     `json:"type"`
	OutputIndex int        `json:"output_index"`
	Item        OutputItem `json:"item"`
}

// SSEResponseDone is the payload for response.done events.
// Only the Status field is used; Output is omitted to avoid deserializing it.
type SSEResponseDone struct {
	Type     string `json:"type"`
	Response struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		ConversationID string `json:"conversation_id"`
	} `json:"response"`
}

// SSEError is the payload for error events.
// The server may use "code" or "error_code" for the error code field.
type SSEError struct {
	Code      string `json:"code"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// ErrorCodeString returns whichever error code field the server populated.
func (e SSEError) ErrorCodeString() string {
	if e.ErrorCode != "" {
		return e.ErrorCode
	}
	return e.Code
}

// SSEEventEnvelope is used for initial type detection of SSE JSON payloads.
type SSEEventEnvelope struct {
	Type string `json:"type"`
}
