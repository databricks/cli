# Step 3: MCP Protocol Implementation

## Overview
Implement the Model Context Protocol (MCP) server infrastructure including protocol handling, tool registration, and request/response processing.

## Tasks

### 3.1 Research MCP SDK Options

Evaluate available options:
1. Use existing Go MCP SDK if available
2. Port minimal MCP implementation from TypeScript reference
3. Use STDIO transport for Claude Code integration

Decision: Check for `github.com/mark3labs/mcp-go` or similar, otherwise implement minimal protocol.

### 3.2 Define MCP Protocol Types

**pkg/mcp/types.go:**

```go
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *ErrorData  `json:"error,omitempty"`
}

type ErrorData struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type Tool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}

type ToolResult struct {
    Content []Content `json:"content"`
    IsError bool      `json:"isError,omitempty"`
}

type Content struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}
```

### 3.3 Implement STDIO Transport

**pkg/mcp/transport/stdio.go:**

```go
type StdioTransport struct {
    reader  *bufio.Reader
    writer  *bufio.Writer
    mu      sync.Mutex
}

func NewStdioTransport() *StdioTransport {
    return &StdioTransport{
        reader: bufio.NewReader(os.Stdin),
        writer: bufio.NewWriter(os.Stdout),
    }
}

func (t *StdioTransport) Read() (*Request, error) {
    line, err := t.reader.ReadBytes('\n')
    if err != nil {
        return nil, err
    }

    var req Request
    if err := json.Unmarshal(line, &req); err != nil {
        return nil, err
    }

    return &req, nil
}

func (t *StdioTransport) Write(resp *Response) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    data, err := json.Marshal(resp)
    if err != nil {
        return err
    }

    if _, err := t.writer.Write(data); err != nil {
        return err
    }

    if err := t.writer.WriteByte('\n'); err != nil {
        return err
    }

    return t.writer.Flush()
}
```

### 3.4 Create Server Handler Interface

**pkg/mcp/handler.go:**

```go
type ServerInfo struct {
    Name         string            `json:"name"`
    Version      string            `json:"version"`
    Capabilities map[string]bool   `json:"capabilities"`
}

type Handler interface {
    // GetInfo returns server metadata
    GetInfo() ServerInfo

    // ListTools returns all available tools
    ListTools(ctx context.Context) ([]Tool, error)

    // CallTool executes a tool with given parameters
    CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error)
}
```

### 3.5 Implement MCP Server

**pkg/mcp/server.go:**

```go
type Server struct {
    handler   Handler
    transport Transport
    logger    *slog.Logger
    shutdown  chan struct{}
}

func NewServer(handler Handler, transport Transport, logger *slog.Logger) *Server {
    return &Server{
        handler:   handler,
        transport: transport,
        logger:    logger,
        shutdown:  make(chan struct{}),
    }
}

func (s *Server) Serve(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-s.shutdown:
            return nil
        default:
            if err := s.handleRequest(ctx); err != nil {
                if err == io.EOF {
                    return nil
                }
                s.logger.Error("request handling error", "error", err)
            }
        }
    }
}

func (s *Server) handleRequest(ctx context.Context) error {
    req, err := s.transport.Read()
    if err != nil {
        return err
    }

    s.logger.Debug("received request", "method", req.Method, "id", req.ID)

    resp := s.dispatch(ctx, req)
    return s.transport.Write(resp)
}

func (s *Server) dispatch(ctx context.Context, req *Request) *Response {
    switch req.Method {
    case "initialize":
        return s.handleInitialize(ctx, req)
    case "tools/list":
        return s.handleListTools(ctx, req)
    case "tools/call":
        return s.handleCallTool(ctx, req)
    default:
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &ErrorData{
                Code:    -32601,
                Message: fmt.Sprintf("method not found: %s", req.Method),
            },
        }
    }
}

func (s *Server) handleInitialize(ctx context.Context, req *Request) *Response {
    info := s.handler.GetInfo()
    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  info,
    }
}

func (s *Server) handleListTools(ctx context.Context, req *Request) *Response {
    tools, err := s.handler.ListTools(ctx)
    if err != nil {
        return s.errorResponse(req.ID, -32603, err.Error())
    }

    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  map[string]interface{}{"tools": tools},
    }
}

func (s *Server) handleCallTool(ctx context.Context, req *Request) *Response {
    var params struct {
        Name      string          `json:"name"`
        Arguments json.RawMessage `json:"arguments"`
    }

    if err := json.Unmarshal(req.Params, &params); err != nil {
        return s.errorResponse(req.ID, -32602, "invalid params")
    }

    result, err := s.handler.CallTool(ctx, params.Name, params.Arguments)
    if err != nil {
        return s.errorResponse(req.ID, -32603, err.Error())
    }

    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  result,
    }
}

func (s *Server) errorResponse(id interface{}, code int, message string) *Response {
    return &Response{
        JSONRPC: "2.0",
        ID:      id,
        Error: &ErrorData{
            Code:    code,
            Message: message,
        },
    }
}
```

### 3.6 Implement Session Management

**pkg/session/session.go:**

```go
type Session struct {
    ID          string
    WorkDir     string
    mu          sync.RWMutex
    startTime   time.Time
    firstTool   bool
    toolCalls   int
}

func NewSession() *Session {
    return &Session{
        ID:        generateID(),
        startTime: time.Now(),
        firstTool: true,
    }
}

func (s *Session) SetWorkDir(dir string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.WorkDir != "" {
        return errors.New("work directory already set")
    }

    s.WorkDir = dir
    return nil
}

func (s *Session) GetWorkDir() (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if s.WorkDir == "" {
        return "", errors.New("work directory not set")
    }

    return s.WorkDir, nil
}

func (s *Session) IsFirstTool() bool {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.firstTool {
        s.firstTool = false
        return true
    }
    return false
}

func generateID() string {
    return fmt.Sprintf("%d-%s", time.Now().Unix(), randomString(8))
}
```

### 3.7 Create Tool Registry

**pkg/mcp/registry.go:**

```go
type ToolFunc func(ctx context.Context, params json.RawMessage) (*ToolResult, error)

type Registry struct {
    tools map[string]*ToolEntry
    mu    sync.RWMutex
}

type ToolEntry struct {
    Tool Tool
    Func ToolFunc
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]*ToolEntry),
    }
}

func (r *Registry) Register(tool Tool, fn ToolFunc) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.tools[tool.Name]; exists {
        return fmt.Errorf("tool already registered: %s", tool.Name)
    }

    r.tools[tool.Name] = &ToolEntry{
        Tool: tool,
        Func: fn,
    }

    return nil
}

func (r *Registry) ListTools() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tools := make([]Tool, 0, len(r.tools))
    for _, entry := range r.tools {
        tools = append(tools, entry.Tool)
    }

    return tools
}

func (r *Registry) Call(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
    r.mu.RLock()
    entry, exists := r.tools[name]
    r.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("tool not found: %s", name)
    }

    return entry.Func(ctx, params)
}
```

### 3.8 Write Tests

**pkg/mcp/server_test.go:**

```go
func TestServer_Initialize(t *testing.T)
func TestServer_ListTools(t *testing.T)
func TestServer_CallTool(t *testing.T)
func TestServer_InvalidMethod(t *testing.T)
func TestServer_InvalidParams(t *testing.T)

// Mock transport for testing
type MockTransport struct {
    requests  []*Request
    responses []*Response
    readIdx   int
}

func (m *MockTransport) Read() (*Request, error) {
    if m.readIdx >= len(m.requests) {
        return nil, io.EOF
    }
    req := m.requests[m.readIdx]
    m.readIdx++
    return req, nil
}

func (m *MockTransport) Write(resp *Response) error {
    m.responses = append(m.responses, resp)
    return nil
}
```

**pkg/mcp/registry_test.go:**

```go
func TestRegistry_Register(t *testing.T)
func TestRegistry_DuplicateRegistration(t *testing.T)
func TestRegistry_ListTools(t *testing.T)
func TestRegistry_Call(t *testing.T)
func TestRegistry_CallNonExistent(t *testing.T)
```

## Acceptance Criteria

- [ ] MCP protocol types defined
- [ ] STDIO transport working
- [ ] Server handles initialize, list_tools, call_tool
- [ ] Tool registry supports registration and lookup
- [ ] Session management tracks state
- [ ] Error responses follow JSON-RPC 2.0 spec
- [ ] All unit tests pass
- [ ] Integration test with mock client succeeds

## Testing Commands

```bash
# Run MCP tests
go test ./pkg/mcp/...

# Test with mock data
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | go run ./cmd/go-mcp

# Integration test
go test ./test/integration/mcp_test.go
```

## Next Steps

Proceed to Step 4: Databricks Provider once all acceptance criteria are met.
