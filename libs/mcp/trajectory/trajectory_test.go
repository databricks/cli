package trajectory

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterWriteEntry(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)
	defer writer.Close()

	entry := NewSessionEntry("test-session", map[string]interface{}{
		"allow_deployment": true,
	})

	err = writer.WriteEntry(entry)
	require.NoError(t, err)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "test-session")
	assert.Contains(t, string(data), "session")

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 1)
}

func TestTypesSessionEntry(t *testing.T) {
	entry := NewSessionEntry("test-session", map[string]interface{}{
		"key": "value",
	})

	assert.Equal(t, EntryTypeSession, entry.EntryType)
	assert.NotNil(t, entry.Session)
	assert.Nil(t, entry.Tool)
	assert.Equal(t, "test-session", entry.Session.SessionID)
	assert.Equal(t, "value", entry.Session.Config["key"])
}

func TestTypesToolEntry(t *testing.T) {
	args := json.RawMessage(`{"param": "value"}`)
	result := json.RawMessage(`{"output": "success"}`)
	errMsg := "test error"

	entry := NewToolEntry("test-session", "test_tool", &args, false, &result, &errMsg)

	assert.Equal(t, EntryTypeTool, entry.EntryType)
	assert.NotNil(t, entry.Tool)
	assert.Nil(t, entry.Session)
	assert.Equal(t, "test-session", entry.Tool.SessionID)
	assert.Equal(t, "test_tool", entry.Tool.ToolName)
	assert.False(t, entry.Tool.Success)
	assert.Equal(t, "test error", *entry.Tool.Error)
}

func TestJSONLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)
	defer writer.Close()

	sessionEntry := NewSessionEntry("sess-1", map[string]interface{}{"test": true})
	err = writer.WriteEntry(sessionEntry)
	require.NoError(t, err)

	args := json.RawMessage(`{"key": "value"}`)
	result := json.RawMessage(`{"result": "ok"}`)
	toolEntry := NewToolEntry("sess-1", "tool1", &args, true, &result, nil)
	err = writer.WriteEntry(toolEntry)
	require.NoError(t, err)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)

	var firstEntry HistoryEntry
	err = json.Unmarshal([]byte(lines[0]), &firstEntry)
	require.NoError(t, err)
	assert.Equal(t, EntryTypeSession, firstEntry.EntryType)

	var secondEntry HistoryEntry
	err = json.Unmarshal([]byte(lines[1]), &secondEntry)
	require.NoError(t, err)
	assert.Equal(t, EntryTypeTool, secondEntry.EntryType)
}

func TestWriterConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)
	defer writer.Close()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			entry := NewSessionEntry("session", map[string]interface{}{"id": id})
			writer.WriteEntry(entry)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 10)
}

func TestTrackerNewTracker(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	sess := &session.Session{ID: "test-session-id"}
	cfg := &mcp.Config{
		AllowDeployment:    true,
		WithWorkspaceTools: true,
		WarehouseID:        "warehouse-123",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tracker, err := NewTracker(sess, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, tracker)
	defer tracker.Close()

	assert.Equal(t, "test-session-id", tracker.sessionID)
	assert.True(t, tracker.enabled)

	historyPath := filepath.Join(tmpDir, ".go-mcp", "history.jsonl")
	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-session-id")
}

func TestTrackerRecordToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   true,
		sessionID: "test-session",
	}
	defer tracker.Close()

	args := map[string]string{"key": "value"}
	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "success"}},
	}

	tracker.RecordToolCall("test_tool", args, result, nil)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test_tool")
	assert.Contains(t, string(data), "test-session")
}

func TestTrackerRecordToolCallWithError(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   true,
		sessionID: "test-session",
	}
	defer tracker.Close()

	args := map[string]string{"param": "test"}
	testErr := assert.AnError

	tracker.RecordToolCall("failing_tool", args, nil, testErr)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "failing_tool")
	assert.Contains(t, string(data), assert.AnError.Error())
}

func TestTrackerClose(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		enabled:   true,
		sessionID: "test",
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	err = tracker.Close()
	assert.NoError(t, err)

	tracker.enabled = false
	err = tracker.Close()
	assert.NoError(t, err)
}

func TestWrapToolHandlerWithTrajectory(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   true,
		sessionID: "test-session",
	}
	defer tracker.Close()

	type TestArgs struct {
		Input string
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest, args TestArgs) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "processed"}},
		}, nil, nil
	}

	wrapped := WrapToolHandlerWithTrajectory(tracker, "wrapped_tool", handler)

	result, data, err := wrapped(nil, &mcp.CallToolRequest{}, TestArgs{Input: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, data)

	histData, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(histData), "wrapped_tool")
}

func TestWrapToolHandlerWithNilTracker(t *testing.T) {
	type TestArgs struct {
		Input string
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest, args TestArgs) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "success"}},
		}, "data", nil
	}

	wrapped := WrapToolHandlerWithTrajectory(nil, "test_tool", handler)

	result, data, err := wrapped(nil, &mcp.CallToolRequest{}, TestArgs{Input: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "data", data)
}

func TestWriterNewWriterError(t *testing.T) {
	writer, err := NewWriter("/nonexistent/path/to/history.jsonl")
	assert.Error(t, err)
	assert.Nil(t, writer)
}

func TestTrackerWriteSessionEntryWithIoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   true,
		sessionID: "test-session",
	}
	defer tracker.Close()

	cfg := &mcp.Config{
		AllowDeployment:    false,
		WithWorkspaceTools: false,
		DatabricksHost:     "https://example.databricks.com",
		WarehouseID:        "warehouse-xyz",
		IoConfig: &mcp.IoConfig{
			Template: &config.TemplateConfig{
				Name: "TRPC",
			},
			Validation: &mcp.ValidationConfig{
				Command:   "npm test",
				UseDagger: true,
			},
		},
	}

	err = tracker.writeSessionEntry(cfg)
	assert.NoError(t, err)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-session")
	assert.Contains(t, string(data), "***")
}

func TestTrackerRecordToolCallWithNilArgs(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   true,
		sessionID: "test-session",
	}
	defer tracker.Close()

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "result"}},
	}

	tracker.RecordToolCall("test_tool", nil, result, nil)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test_tool")
}

func TestTrackerRecordToolCallDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   false,
		sessionID: "test-session",
	}
	defer tracker.Close()

	tracker.RecordToolCall("test_tool", map[string]string{"key": "value"}, nil, nil)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "test_tool")
}

func TestWrapToolHandlerWithDisabledTracker(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	writer, err := NewWriter(historyPath)
	require.NoError(t, err)

	tracker := &Tracker{
		writer:    writer,
		session:   &session.Session{ID: "test-session"},
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		enabled:   false,
		sessionID: "test-session",
	}
	defer tracker.Close()

	type TestArgs struct {
		Input string
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest, args TestArgs) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "processed"}},
		}, nil, nil
	}

	wrapped := WrapToolHandlerWithTrajectory(tracker, "wrapped_tool", handler)

	result, _, err := wrapped(nil, &mcp.CallToolRequest{}, TestArgs{Input: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "wrapped_tool")
}
