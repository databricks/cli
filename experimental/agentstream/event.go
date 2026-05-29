package agentstream

// EventKind identifies the type of stream event.
type EventKind int

const (
	EventThinking EventKind = iota // status line on stderr
	EventText                      // markdown to stdout
	EventToolCall                  // function call (e.g. execute_sql)
	EventError                     // API error
	EventDone                      // stream completed
	EventViz                       // visualization chart
)

// StreamEvent is the protocol-agnostic unit that renderers consume.
// Protocol-specific adapters convert raw SSE data into these.
type StreamEvent struct {
	Kind      EventKind
	Text      string         // for Thinking, Text, Error
	ToolCall  *ToolCallEvent // for ToolCall
	Viz       *VizEvent      // for Viz
	Status    string         // for Done ("completed", "failed")
	ErrorCode string         // for Error
	Raw       string         // original SSE data (for debug)
}

// VizEvent carries visualization data for terminal chart rendering.
type VizEvent struct {
	Spec *VizSpec
	Data *TableData
}

// VizSpec describes how to render a visualization.
type VizSpec struct {
	Title      string
	WidgetType string // "bar", "line", "area"
	XField     string
	YFields    []string
	ColorField string
	Layout     string // "stack", "group"
	XTitle     string
	YTitle     string
}

// TableData holds parsed tabular data.
type TableData struct {
	Columns []string
	Rows    [][]string
}

// ToolCallEvent represents a function call emitted by an agent.
type ToolCallEvent struct {
	Name      string
	Arguments string
}

// AdapterFunc converts a raw SSE data payload into zero or more StreamEvents.
// Each protocol (OneChat, ChatCompletions, Anthropic) implements one of these.
type AdapterFunc func(data string) []StreamEvent

// RenderOptions controls what RenderText displays.
type RenderOptions struct {
	ShowSQL bool // display SQL queries executed by the agent
}

// StreamResult is the structured output for --output json mode.
type StreamResult struct {
	Status    string     `json:"status"`
	Text      string     `json:"text,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is a simplified function call for JSON output.
type ToolCall struct {
	Name      string `json:"name"`
	SQL       string `json:"sql,omitempty"`
	Title     string `json:"title,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
