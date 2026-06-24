package genie

// GenieRequest is the request body for POST /api/2.0/data-rooms/tools/onechat/responses.
type GenieRequest struct {
	Input       []InputItem `json:"input"`
	WarehouseID string      `json:"warehouseId,omitempty"`
	// ConversationID continues an existing conversation, carrying the
	// conversation_id from a prior response. Empty starts a new one. The request
	// field is camelCase (matching warehouseId); the snake_case conversation_id
	// the response uses is NOT read back on the request — verified live.
	ConversationID string `json:"conversationId,omitempty"`
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
	Type      string        `json:"type"`
	ID        string        `json:"id,omitempty"`
	Role      string        `json:"role,omitempty"`
	Status    string        `json:"status,omitempty"`
	Name      string        `json:"name,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	Content   []ContentItem `json:"content,omitempty"`
	// Metadata values are not all strings: the backend sends arrays (e.g.
	// source_internal_ids) and objects, so this must be map[string]any or the
	// whole item fails to unmarshal and the message/tool call is silently lost.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// UIType returns the metadata ui_type field (e.g. "THOUGHT", "FINAL_RESPONSE").
func (o OutputItem) UIType() string {
	s, _ := o.Metadata["ui_type"].(string)
	return s
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
		Type     string             `json:"type"`
		ID       string             `json:"id"`
		Status   string             `json:"status"`
		Output   string             `json:"output"`
		Metadata funcCallOutputMeta `json:"metadata"`
	} `json:"item"`
}

// funcCallOutputMeta holds metadata from function_call_output items.
type funcCallOutputMeta struct {
	UIType      string `json:"ui_type"`
	StatementID string `json:"statement_id"`
	SQLID       string `json:"sql_id"`
	EmbedID     string `json:"embed_id"`
	// VizDefinition is a JSON-encoded Helios chart spec (the spec itself is
	// nested under "renderSpec"). The backend sends it as a string, not an
	// object, and it carries no data.
	VizDefinition string `json:"viz_definition"`
	// ResultData carries the SQL preview rows for QUERY_EXECUTION outputs. A
	// later viz is joined to this data by statement_id.
	ResultData *queryResultData `json:"result_data"`
}

// heliosVizDefinition is the parsed metadata.viz_definition payload; the Helios
// chart spec is nested under renderSpec.
type heliosVizDefinition struct {
	RenderSpec *heliosSpec `json:"renderSpec"`
}

// heliosSpec is the Helios chart specification: a widget type plus field
// encodings. It carries no data; the rows live in the matching QUERY_EXECUTION
// result_data, joined by statement_id.
type heliosSpec struct {
	WidgetType string          `json:"widgetType"`
	Frame      heliosFrame     `json:"frame"`
	Encodings  heliosEncodings `json:"encodings"`
}

type heliosFrame struct {
	Title string `json:"title"`
}

type heliosEncodings struct {
	X *heliosEncoding `json:"x"`
	Y *heliosEncoding `json:"y"`
}

// heliosEncoding describes one axis. A single field uses FieldName; multiple
// series use Fields. DisplayName is the axis label when present.
type heliosEncoding struct {
	FieldName   string        `json:"fieldName"`
	DisplayName string        `json:"displayName"`
	Fields      []heliosField `json:"fields"`
}

type heliosField struct {
	FieldName string `json:"fieldName"`
}

// queryResultData is the QUERY_EXECUTION result_data: column metadata plus
// preview rows in array-of-arrays wire format.
type queryResultData struct {
	Columns     []queryResultColumn `json:"columns"`
	PreviewRows [][]any             `json:"preview_rows"`
}

type queryResultColumn struct {
	Name string `json:"name"`
}
