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

// funcCallOutputEvent is used to parse function_call_output items from SSE events.
// It uses json.RawMessage for metadata since viz events contain nested objects.
type funcCallOutputEvent struct {
	Item struct {
		Type     string          `json:"type"`
		ID       string          `json:"id"`
		Status   string          `json:"status"`
		Output   string          `json:"output"`
		Metadata funcCallOutputMeta `json:"metadata"`
	} `json:"item"`
}

// funcCallOutputMeta holds metadata from function_call_output items.
type funcCallOutputMeta struct {
	UIType      string          `json:"ui_type"`
	StatementID string          `json:"statement_id"`
	SQLID       string          `json:"sql_id"`
	EmbedID     string          `json:"embed_id"`
	RenderSpec  *renderSpecJSON `json:"render_spec"`
}

// renderSpecJSON is the Lakeview-style visualization specification.
type renderSpecJSON struct {
	WidgetType string          `json:"widgetType"`
	Frame      renderSpecFrame `json:"frame"`
	Encodings  renderSpecEnc   `json:"encodings"`
	Mark       renderSpecMark  `json:"mark"`
}

type renderSpecFrame struct {
	Title     string `json:"title"`
	ShowTitle bool   `json:"showTitle"`
}

type renderSpecEnc struct {
	X     *renderSpecField `json:"x"`
	Y     *renderSpecField `json:"y"`
	Color *renderSpecField `json:"color"`
}

type renderSpecField struct {
	FieldName string `json:"fieldName"`
	Axis      struct {
		Title string `json:"title"`
	} `json:"axis"`
}

type renderSpecMark struct {
	Layout string `json:"layout"`
}
