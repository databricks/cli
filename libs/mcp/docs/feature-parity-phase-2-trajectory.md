# Phase 2: Trajectory Tracking Implementation

## Objective

Implement JSONL-based history logging system to track all MCP tool calls for debugging, analytics, and replay capabilities. This provides crucial observability into agent behavior and workflow patterns.

## Priority

**🟡 MEDIUM** - High value for debugging and analytics, but not blocking core functionality.

## Context

The Rust implementation at `edda_mcp/src/trajectory.rs` provides the reference implementation. This feature logs all tool calls and results to `~/.app-mcp/history.jsonl` for later analysis.

**Benefits:**
- **Debugging**: Replay failed workflows to identify issues
- **Analytics**: Understand tool usage patterns
- **Audit Trail**: Track what operations were performed
- **Testing**: Use real trajectories for integration tests

## Prerequisites

- JSON serialization (Go stdlib)
- File I/O and rotation utilities
- Study Rust implementation: `/Users/fabian.jakobs/Workspaces/agent/edda/edda_mcp/src/trajectory.rs`
- Understand JSONL format (JSON Lines)

## Implementation Steps

### Step 1: Create Trajectory Package Structure

Create `pkg/trajectory/` directory with the following files:
- `trajectory.go` - Main tracker implementation
- `writer.go` - JSONL file writer with rotation
- `types.go` - Event type definitions
- `query.go` - Query interface for reading history
- `trajectory_test.go` - Unit tests

### Step 2: Define Event Types

#### types.go

```go
package trajectory

import (
    "encoding/json"
    "time"
)

// EventType represents the type of trajectory event
type EventType string

const (
    EventTypeSessionStart EventType = "session_start"
    EventTypeToolCall     EventType = "tool_call"
)

// BaseEvent contains fields common to all events
type BaseEvent struct {
    SessionID string    `json:"session_id"`
    Timestamp time.Time `json:"timestamp"`
    EventType EventType `json:"event_type"`
}

// SessionStartEvent is logged once at the beginning of each session
type SessionStartEvent struct {
    BaseEvent
    Config map[string]interface{} `json:"config"` // Redacted sensitive fields
    Version string                `json:"version"`
    OS      string                `json:"os"`
    Arch    string                `json:"arch"`
}

// ToolCallEvent tracks individual tool invocations
type ToolCallEvent struct {
    BaseEvent
    ToolName  string                 `json:"tool_name"`
    Arguments map[string]interface{} `json:"arguments"` // Redacted sensitive fields
    Success   bool                   `json:"success"`
    Duration  time.Duration          `json:"duration_ms"` // in milliseconds
    Result    json.RawMessage        `json:"result,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

// Event is a union type for all event types
type Event struct {
    *SessionStartEvent
    *ToolCallEvent
}

// MarshalJSON implements custom JSON marshaling
func (e Event) MarshalJSON() ([]byte, error) {
    if e.SessionStartEvent != nil {
        return json.Marshal(e.SessionStartEvent)
    }
    if e.ToolCallEvent != nil {
        return json.Marshal(e.ToolCallEvent)
    }
    return nil, fmt.Errorf("empty event")
}
```

### Step 3: Implement JSONL Writer

#### writer.go

```go
package trajectory

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
)

// Writer writes trajectory events to a JSONL file
type Writer struct {
    file *os.File
    mu   sync.Mutex
}

// NewWriter creates a new trajectory writer
// File location: ~/.app-mcp/history.jsonl
func NewWriter() (*Writer, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("failed to get home directory: %w", err)
    }

    mcpDir := filepath.Join(homeDir, ".app-mcp")
    if err := os.MkdirAll(mcpDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create .app-mcp directory: %w", err)
    }

    historyFile := filepath.Join(mcpDir, "history.jsonl")

    file, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open history file: %w", err)
    }

    return &Writer{
        file: file,
    }, nil
}

// WriteEvent writes an event to the JSONL file
func (w *Writer) WriteEvent(event Event) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    // Write JSON line
    if _, err := w.file.Write(data); err != nil {
        return fmt.Errorf("failed to write event: %w", err)
    }

    // Write newline
    if _, err := w.file.Write([]byte("\n")); err != nil {
        return fmt.Errorf("failed to write newline: %w", err)
    }

    // Flush to ensure write completes
    return w.file.Sync()
}

// Close closes the writer
func (w *Writer) Close() error {
    w.mu.Lock()
    defer w.mu.Unlock()

    if w.file != nil {
        return w.file.Close()
    }
    return nil
}

// Rotate rotates the history file (optional)
// Creates history.jsonl.1, history.jsonl.2, etc.
func (w *Writer) Rotate(maxFiles int) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    // Close current file
    if err := w.file.Close(); err != nil {
        return err
    }

    homeDir, _ := os.UserHomeDir()
    historyFile := filepath.Join(homeDir, ".app-mcp", "history.jsonl")

    // Rotate existing files
    for i := maxFiles - 1; i >= 1; i-- {
        oldPath := fmt.Sprintf("%s.%d", historyFile, i)
        newPath := fmt.Sprintf("%s.%d", historyFile, i+1)

        if _, err := os.Stat(oldPath); err == nil {
            os.Rename(oldPath, newPath)
        }
    }

    // Rename current file
    os.Rename(historyFile, historyFile+".1")

    // Open new file
    file, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }

    w.file = file
    return nil
}
```

### Step 4: Implement Trajectory Tracker

#### trajectory.go

```go
package trajectory

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "runtime"
    "sync"
    "time"

    "app-mcp/pkg/config"
    "app-mcp/pkg/session"
    "app-mcp/pkg/version"
)

// Tracker tracks trajectory events
type Tracker struct {
    writer    *Writer
    session   *session.Session
    config    *config.Config
    enabled   bool
    mu        sync.RWMutex
    eventChan chan Event
    wg        sync.WaitGroup
}

// NewTracker creates a new trajectory tracker
// Reference: edda_mcp/src/trajectory.rs:25-55
func NewTracker(sess *session.Session, cfg *config.Config) (*Tracker, error) {
    // Only enable in production (not during development)
    enabled := os.Getenv("CARGO") == "" // Rust checks this
    // For Go, check if we're running from go run
    if os.Getenv("GO_MCP_DEV") != "" {
        enabled = false
    }

    // Allow explicit override
    if os.Getenv("GO_MCP_TRAJECTORY_ENABLED") == "true" {
        enabled = true
    } else if os.Getenv("GO_MCP_TRAJECTORY_ENABLED") == "false" {
        enabled = false
    }

    if !enabled {
        return &Tracker{enabled: false}, nil
    }

    writer, err := NewWriter()
    if err != nil {
        return nil, fmt.Errorf("failed to create trajectory writer: %w", err)
    }

    tracker := &Tracker{
        writer:    writer,
        session:   sess,
        config:    cfg,
        enabled:   true,
        eventChan: make(chan Event, 100), // Buffered channel
    }

    // Start background writer
    tracker.wg.Add(1)
    go tracker.eventWriter()

    // Write session start event
    tracker.logSessionStart()

    return tracker, nil
}

// eventWriter processes events in the background
func (t *Tracker) eventWriter() {
    defer t.wg.Done()

    for event := range t.eventChan {
        if err := t.writer.WriteEvent(event); err != nil {
            // Log error but don't fail
            fmt.Fprintf(os.Stderr, "trajectory: failed to write event: %v\n", err)
        }
    }
}

// logSessionStart writes the session start event
func (t *Tracker) logSessionStart() {
    if !t.enabled {
        return
    }

    event := Event{
        SessionStartEvent: &SessionStartEvent{
            BaseEvent: BaseEvent{
                SessionID: t.session.ID(),
                Timestamp: time.Now(),
                EventType: EventTypeSessionStart,
            },
            Config:  t.redactConfig(),
            Version: version.Version,
            OS:      runtime.GOOS,
            Arch:    runtime.GOARCH,
        },
    }

    t.eventChan <- event
}

// LogToolCall logs a tool invocation
// Reference: edda_mcp/src/trajectory.rs:57-85
func (t *Tracker) LogToolCall(
    toolName string,
    arguments map[string]interface{},
    success bool,
    duration time.Duration,
    result interface{},
    err error,
) {
    if !t.enabled {
        return
    }

    event := ToolCallEvent{
        BaseEvent: BaseEvent{
            SessionID: t.session.ID(),
            Timestamp: time.Now(),
            EventType: EventTypeToolCall,
        },
        ToolName:  toolName,
        Arguments: t.redactArguments(arguments),
        Success:   success,
        Duration:  duration,
    }

    if success && result != nil {
        resultJSON, _ := json.Marshal(result)
        event.Result = resultJSON
    }

    if err != nil {
        event.Error = err.Error()
    }

    t.eventChan <- Event{ToolCallEvent: &event}
}

// redactConfig removes sensitive information from config
func (t *Tracker) redactConfig() map[string]interface{} {
    redacted := make(map[string]interface{})

    redacted["allow_deployment"] = t.config.AllowDeployment
    redacted["with_workspace_tools"] = t.config.WithWorkspaceTools

    // Redact sensitive fields
    if t.config.WarehouseID != "" {
        redacted["warehouse_id"] = "***"
    }
    if t.config.DatabricksHost != "" {
        redacted["databricks_host"] = "***"
    }

    return redacted
}

// redactArguments removes sensitive information from arguments
func (t *Tracker) redactArguments(args map[string]interface{}) map[string]interface{} {
    redacted := make(map[string]interface{})

    sensitiveKeys := map[string]bool{
        "token":    true,
        "password": true,
        "secret":   true,
        "key":      true,
        "auth":     true,
    }

    for key, value := range args {
        if sensitiveKeys[key] {
            redacted[key] = "***"
        } else {
            redacted[key] = value
        }
    }

    return redacted
}

// Close closes the tracker
func (t *Tracker) Close() error {
    if !t.enabled {
        return nil
    }

    close(t.eventChan)
    t.wg.Wait()

    return t.writer.Close()
}
```

### Step 5: Integrate with MCP Server

Update `pkg/mcp/server.go` to wrap tool calls:

```go
type Server struct {
    mcpServer *mcp.Server
    session   *session.Session
    logger    *slog.Logger
    tracker   *trajectory.Tracker // Add tracker
}

func NewServer(cfg *config.Config, sess *session.Session, logger *slog.Logger) (*Server, error) {
    // ... existing code ...

    // Create trajectory tracker
    tracker, err := trajectory.NewTracker(sess, cfg)
    if err != nil {
        logger.Warn("failed to create trajectory tracker", "error", err)
        // Continue without tracker
    }

    return &Server{
        mcpServer: mcpServer,
        session:   sess,
        logger:    logger,
        tracker:   tracker,
    }, nil
}

// WrapToolHandler wraps a tool handler with trajectory tracking
func (s *Server) WrapToolHandler(
    toolName string,
    handler func(context.Context, map[string]interface{}) (*mcp.CallToolResult, error),
) func(context.Context, map[string]interface{}) (*mcp.CallToolResult, error) {

    return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
        start := time.Now()

        // Call original handler
        result, err := handler(ctx, args)

        // Track trajectory
        if s.tracker != nil {
            s.tracker.LogToolCall(
                toolName,
                args,
                err == nil,
                time.Since(start),
                result,
                err,
            )
        }

        return result, err
    }
}

// Close cleanup
func (s *Server) Close() error {
    if s.tracker != nil {
        return s.tracker.Close()
    }
    return nil
}
```

### Step 6: Query Interface (Optional)

#### query.go

```go
package trajectory

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// Query allows reading and filtering trajectory history
type Query struct {
    file *os.File
}

// NewQuery opens the history file for reading
func NewQuery() (*Query, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    historyFile := filepath.Join(homeDir, ".app-mcp", "history.jsonl")

    file, err := os.Open(historyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to open history: %w", err)
    }

    return &Query{file: file}, nil
}

// FilterOptions for querying events
type FilterOptions struct {
    SessionID string
    ToolName  string
    Success   *bool
    Limit     int
}

// ReadEvents reads and filters events
func (q *Query) ReadEvents(opts FilterOptions) ([]Event, error) {
    var events []Event

    scanner := bufio.NewScanner(q.file)
    count := 0

    for scanner.Scan() {
        line := scanner.Text()
        if strings.TrimSpace(line) == "" {
            continue
        }

        var event Event
        if err := json.Unmarshal([]byte(line), &event); err != nil {
            continue // Skip malformed lines
        }

        // Apply filters
        if opts.SessionID != "" {
            var sessionID string
            if event.SessionStartEvent != nil {
                sessionID = event.SessionStartEvent.SessionID
            } else if event.ToolCallEvent != nil {
                sessionID = event.ToolCallEvent.SessionID
            }

            if sessionID != opts.SessionID {
                continue
            }
        }

        if opts.ToolName != "" && event.ToolCallEvent != nil {
            if event.ToolCallEvent.ToolName != opts.ToolName {
                continue
            }
        }

        if opts.Success != nil && event.ToolCallEvent != nil {
            if event.ToolCallEvent.Success != *opts.Success {
                continue
            }
        }

        events = append(events, event)
        count++

        if opts.Limit > 0 && count >= opts.Limit {
            break
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return events, nil
}

// Close closes the query
func (q *Query) Close() error {
    return q.file.Close()
}
```

### Step 7: CLI Command (Optional)

Add command to `cmd/app-mcp/cli.go`:

```go
var historyCmd = &cobra.Command{
    Use:   "history",
    Short: "Query trajectory history",
    RunE: func(cmd *cobra.Command, args []string) error {
        sessionID, _ := cmd.Flags().GetString("session")
        toolName, _ := cmd.Flags().GetString("tool")
        limit, _ := cmd.Flags().GetInt("limit")

        query, err := trajectory.NewQuery()
        if err != nil {
            return err
        }
        defer query.Close()

        events, err := query.ReadEvents(trajectory.FilterOptions{
            SessionID: sessionID,
            ToolName:  toolName,
            Limit:     limit,
        })
        if err != nil {
            return err
        }

        // Print events as JSON
        for _, event := range events {
            data, _ := json.MarshalIndent(event, "", "  ")
            fmt.Println(string(data))
        }

        return nil
    },
}

func init() {
    historyCmd.Flags().String("session", "", "Filter by session ID")
    historyCmd.Flags().String("tool", "", "Filter by tool name")
    historyCmd.Flags().Int("limit", 100, "Maximum events to return")

    rootCmd.AddCommand(historyCmd)
}
```

## Testing

### Unit Tests

```go
func TestWriter_WriteEvent(t *testing.T) {
    tmpDir := t.TempDir()

    // Override home directory for testing
    os.Setenv("HOME", tmpDir)
    defer os.Unsetenv("HOME")

    writer, err := NewWriter()
    if err != nil {
        t.Fatalf("failed to create writer: %v", err)
    }
    defer writer.Close()

    event := Event{
        SessionStartEvent: &SessionStartEvent{
            BaseEvent: BaseEvent{
                SessionID: "test-session",
                Timestamp: time.Now(),
                EventType: EventTypeSessionStart,
            },
            Version: "1.0.0",
        },
    }

    if err := writer.WriteEvent(event); err != nil {
        t.Fatalf("failed to write event: %v", err)
    }

    // Verify file contents
    historyFile := filepath.Join(tmpDir, ".app-mcp", "history.jsonl")
    data, err := os.ReadFile(historyFile)
    if err != nil {
        t.Fatalf("failed to read history: %v", err)
    }

    if !strings.Contains(string(data), "test-session") {
        t.Errorf("expected session ID in history")
    }
}

func TestTracker_LogToolCall(t *testing.T) {
    // Test tool call logging with redaction
}

func TestQuery_ReadEvents(t *testing.T) {
    // Test querying events with filters
}
```

## Configuration

No configuration needed - trajectory tracking is:
- Enabled by default in production builds
- Disabled in development (when `GO_MCP_DEV` is set)
- Can be explicitly controlled via `GO_MCP_TRAJECTORY_ENABLED` env var

## Documentation Updates

### CLAUDE.md

Add new section:

```markdown
### Trajectory Tracking (pkg/trajectory)

**JSONL-based history logging**:

- Logs to `~/.app-mcp/history.jsonl`
- Tracks all tool calls with arguments, results, durations
- Session metadata (first entry per session)
- Sensitive data redacted (tokens, passwords, etc.)
- Non-blocking writes via buffered channel
- Query interface for filtering history

**Use Cases:**
- Debugging failed workflows
- Analytics on tool usage patterns
- Replay capability for testing
- Audit trail of operations
```

## Success Criteria

- [ ] Event types defined (SessionStart, ToolCall)
- [ ] JSONL writer implemented with rotation
- [ ] Tracker integrated with MCP server
- [ ] Tool calls automatically logged
- [ ] Sensitive data redacted
- [ ] Non-blocking writes (buffered channel)
- [ ] Query interface working
- [ ] CLI command for history (optional)
- [ ] Unit tests pass
- [ ] Documentation updated

## Timeline

- **Days 1-2**: Event types, writer implementation
- **Days 3-4**: Tracker implementation, MCP integration
- **Days 5-6**: Query interface, CLI command
- **Day 7**: Testing, documentation, polish

## Next Phase

After completing trajectory tracking, proceed to:
- **Phase 3**: Developer Experience Improvements
