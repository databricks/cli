package trajectory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	mcp "github.com/databricks/cli/experimental/aitools/lib"
	mcpsdk "github.com/databricks/cli/experimental/aitools/lib/mcp"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/log"
)

type Tracker struct {
	writer    *Writer
	session   *session.Session
	enabled   bool
	sessionID string
}

func NewTracker(ctx context.Context, sess *session.Session, cfg *mcp.Config) (*Tracker, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyPath := filepath.Join(homeDir, ".databricks", "aitools", "history.jsonl")

	writer, err := NewWriter(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create trajectory writer: %w", err)
	}

	tracker := &Tracker{
		writer:    writer,
		session:   sess,
		enabled:   true,
		sessionID: sess.ID,
	}

	if err := tracker.writeSessionEntry(cfg); err != nil {
		log.Warnf(ctx, "failed to write session entry: %v", err)
	}

	return tracker, nil
}

func (t *Tracker) writeSessionEntry(cfg *mcp.Config) error {
	configMap := make(map[string]any)

	entry := NewSessionEntry(t.sessionID, configMap)
	return t.writer.WriteEntry(entry)
}

func (t *Tracker) RecordToolCall(toolName string, args any, result *mcpsdk.CallToolResult, err error) {
	if !t.enabled {
		return
	}

	var argsJSON *json.RawMessage
	if args != nil {
		data, jsonErr := json.Marshal(args)
		if jsonErr == nil {
			raw := json.RawMessage(data)
			argsJSON = &raw
		}
	}

	var resultJSON *json.RawMessage
	var errorStr *string
	success := err == nil

	if success && result != nil {
		data, jsonErr := json.Marshal(result)
		if jsonErr == nil {
			raw := json.RawMessage(data)
			resultJSON = &raw
		}
	}

	if err != nil {
		errMsg := err.Error()
		errorStr = &errMsg
	}

	entry := NewToolEntry(t.sessionID, toolName, argsJSON, success, resultJSON, errorStr)
	_ = t.writer.WriteEntry(entry)
}

func (t *Tracker) Close() error {
	if !t.enabled {
		return nil
	}
	return t.writer.Close()
}
