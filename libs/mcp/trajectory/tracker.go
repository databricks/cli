package trajectory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Tracker struct {
	writer    *Writer
	session   *session.Session
	logger    *slog.Logger
	enabled   bool
	sessionID string
}

func NewTracker(sess *session.Session, cfg *mcp.Config, logger *slog.Logger) (*Tracker, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyPath := filepath.Join(homeDir, ".go-mcp", "history.jsonl")

	writer, err := NewWriter(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create trajectory writer: %w", err)
	}

	tracker := &Tracker{
		writer:    writer,
		session:   sess,
		logger:    logger,
		enabled:   true,
		sessionID: sess.ID,
	}

	if err := tracker.writeSessionEntry(cfg); err != nil {
		logger.Warn("failed to write session entry", "error", err)
	}

	return tracker, nil
}

func (t *Tracker) writeSessionEntry(cfg *mcp.Config) error {
	configMap := make(map[string]interface{})
	configMap["allow_deployment"] = cfg.AllowDeployment
	configMap["with_workspace_tools"] = cfg.WithWorkspaceTools

	if cfg.WarehouseID != "" {
		configMap["warehouse_id"] = "***"
	}
	if cfg.DatabricksHost != "" {
		configMap["databricks_host"] = "***"
	}

	if cfg.IoConfig != nil {
		ioConfigMap := make(map[string]interface{})
		if cfg.IoConfig.Template != nil {
			ioConfigMap["template"] = fmt.Sprintf("%v", cfg.IoConfig.Template)
		}
		if cfg.IoConfig.Validation != nil {
			ioConfigMap["validation"] = "***"
		}
		configMap["io_config"] = ioConfigMap
	}

	entry := NewSessionEntry(t.sessionID, configMap)
	return t.writer.WriteEntry(entry)
}

func (t *Tracker) RecordToolCall(toolName string, args interface{}, result *mcp.CallToolResult, err error) {
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
	if writeErr := t.writer.WriteEntry(entry); writeErr != nil {
		t.logger.Warn("failed to record trajectory entry", "error", writeErr)
	}
}

func (t *Tracker) Close() error {
	if !t.enabled {
		return nil
	}
	return t.writer.Close()
}

func WrapToolHandlerWithTrajectory[T any](
	tracker *Tracker,
	toolName string,
	handler func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error),
) func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, any, error) {
		result, data, err := handler(ctx, req, args)

		if tracker != nil && tracker.enabled {
			tracker.RecordToolCall(toolName, args, result, err)
		}

		return result, data, err
	}
}
