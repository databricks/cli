package trajectory

import (
	"encoding/json"
	"time"
)

type EntryType string

const (
	EntryTypeSession EntryType = "session"
	EntryTypeTool    EntryType = "tool"
)

type HistoryEntry struct {
	EntryType EntryType     `json:"entry_type"`
	Session   *SessionEntry `json:"session,omitempty"`
	Tool      *ToolEntry    `json:"tool,omitempty"`
}

type SessionEntry struct {
	SessionID string         `json:"session_id"`
	Timestamp string         `json:"timestamp"`
	Config    map[string]any `json:"config"`
}

type ToolEntry struct {
	SessionID string           `json:"session_id"`
	Timestamp string           `json:"timestamp"`
	ToolName  string           `json:"tool_name"`
	Arguments *json.RawMessage `json:"arguments,omitempty"`
	Success   bool             `json:"success"`
	Result    *json.RawMessage `json:"result,omitempty"`
	Error     *string          `json:"error,omitempty"`
}

func NewSessionEntry(sessionID string, config map[string]any) HistoryEntry {
	return HistoryEntry{
		EntryType: EntryTypeSession,
		Session: &SessionEntry{
			SessionID: sessionID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Config:    config,
		},
	}
}

func NewToolEntry(sessionID, toolName string, args *json.RawMessage, success bool, result *json.RawMessage, err *string) HistoryEntry {
	return HistoryEntry{
		EntryType: EntryTypeTool,
		Tool: &ToolEntry{
			SessionID: sessionID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			ToolName:  toolName,
			Arguments: args,
			Success:   success,
			Result:    result,
			Error:     err,
		},
	}
}
