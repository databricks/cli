# Step 7: Integration and Polish

## Overview
Combine all providers into a unified MCP server, add health checks, implement graceful shutdown, and create comprehensive integration tests.

## Tasks

### 7.1 Create Combined Handler

**pkg/server/handler.go:**

```go
type CombinedHandler struct {
    databricks *databricks.Provider
    io         *io.Provider
    workspace  *workspace.Provider
    session    *session.Session
    logger     *slog.Logger
    firstCall  sync.Once
}

func NewCombinedHandler(cfg *config.Config, logger *slog.Logger) (*CombinedHandler, error) {
    // Create session
    sess := session.NewSession()

    // Initialize providers
    var databricksProvider *databricks.Provider
    var ioProvider *io.Provider
    var workspaceProvider *workspace.Provider

    // Databricks (optional)
    if cfg.DatabricksHost != "" && cfg.DatabricksToken != "" {
        db, err := databricks.NewProvider(cfg, logger)
        if err != nil {
            logger.Warn("Databricks provider unavailable", "error", err)
        } else {
            databricksProvider = db
            logger.Info("Databricks provider initialized")
        }
    }

    // I/O provider
    if cfg.IOConfig != nil {
        provider, err := io.NewProvider(cfg.IOConfig, logger)
        if err != nil {
            logger.Warn("I/O provider unavailable", "error", err)
        } else {
            ioProvider = provider
            logger.Info("I/O provider initialized")
        }
    }

    // Workspace provider (if enabled)
    if cfg.WithWorkspaceTools {
        provider, err := workspace.NewProvider(sess, logger)
        if err != nil {
            logger.Warn("Workspace provider unavailable", "error", err)
        } else {
            workspaceProvider = provider
            logger.Info("Workspace provider initialized")
        }
    }

    // Ensure at least one provider is available
    if databricksProvider == nil && ioProvider == nil && workspaceProvider == nil {
        return nil, errors.New(
            "no providers available - configure at least one:\n" +
                "- Databricks: Set DATABRICKS_HOST, DATABRICKS_TOKEN, DATABRICKS_WAREHOUSE_ID\n" +
                "- I/O: Configure io_config in ~/.go-mcp/config.json\n" +
                "- Workspace: Use --with-workspace-tools flag",
        )
    }

    return &CombinedHandler{
        databricks: databricksProvider,
        io:         ioProvider,
        workspace:  workspaceProvider,
        session:    sess,
        logger:     logger,
    }, nil
}

func (h *CombinedHandler) GetInfo() mcp.ServerInfo {
    providers := []string{}
    if h.databricks != nil {
        providers = append(providers, "Databricks")
    }
    if h.io != nil {
        providers = append(providers, "I/O")
    }
    if h.workspace != nil {
        providers = append(providers, "Workspace")
    }

    return mcp.ServerInfo{
        Name:    "go-mcp",
        Version: version.Version,
        Capabilities: map[string]bool{
            "tools": true,
        },
        Instructions: fmt.Sprintf(
            "Go MCP Server providing integrations for: %s",
            strings.Join(providers, ", "),
        ),
    }
}

func (h *CombinedHandler) ListTools(ctx context.Context) ([]mcp.Tool, error) {
    tools := []mcp.Tool{}

    if h.databricks != nil {
        dbTools, err := h.databricks.ListTools(ctx)
        if err != nil {
            h.logger.Warn("failed to list Databricks tools", "error", err)
        } else {
            tools = append(tools, dbTools...)
        }
    }

    if h.io != nil {
        ioTools, err := h.io.ListTools(ctx)
        if err != nil {
            h.logger.Warn("failed to list I/O tools", "error", err)
        } else {
            tools = append(tools, ioTools...)
        }
    }

    if h.workspace != nil {
        wsTools, err := h.workspace.ListTools(ctx)
        if err != nil {
            h.logger.Warn("failed to list Workspace tools", "error", err)
        } else {
            tools = append(tools, wsTools...)
        }
    }

    return tools, nil
}

func (h *CombinedHandler) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.ToolResult, error) {
    // Intercept scaffold_data_app to set work directory
    if name == "scaffold_data_app" && h.io != nil {
        result, err := h.io.CallTool(ctx, name, params)
        if err != nil {
            return nil, err
        }

        // Extract work_dir from params and set in session
        var args struct {
            WorkDir string `json:"work_dir"`
        }
        if err := json.Unmarshal(params, &args); err == nil {
            if err := h.session.SetWorkDir(args.WorkDir); err != nil {
                h.logger.Warn("failed to set work directory", "error", err)
            } else {
                h.logger.Info("work directory set", "dir", args.WorkDir)
            }
        }

        return result, nil
    }

    // Route to appropriate provider based on tool name prefix
    if strings.HasPrefix(name, "databricks_") && h.databricks != nil {
        return h.databricks.CallTool(ctx, name, params)
    }

    if (name == "scaffold_data_app" || name == "validate_data_app") && h.io != nil {
        return h.io.CallTool(ctx, name, params)
    }

    if (name == "read_file" || name == "write_file" || name == "edit_file" ||
        name == "bash" || name == "grep" || name == "glob") && h.workspace != nil {
        return h.workspace.CallTool(ctx, name, params)
    }

    return nil, fmt.Errorf("unknown tool: %s", name)
}
```

### 7.2 Implement Health Checks

**pkg/server/health.go:**

```go
type HealthChecker struct {
    handler *CombinedHandler
    logger  *slog.Logger
}

func NewHealthChecker(handler *CombinedHandler, logger *slog.Logger) *HealthChecker {
    return &HealthChecker{
        handler: handler,
        logger:  logger,
    }
}

type HealthStatus struct {
    Healthy   bool              `json:"healthy"`
    Providers map[string]string `json:"providers"`
    Timestamp time.Time         `json:"timestamp"`
}

func (hc *HealthChecker) Check(ctx context.Context) *HealthStatus {
    status := &HealthStatus{
        Healthy:   true,
        Providers: make(map[string]string),
        Timestamp: time.Now(),
    }

    // Check Databricks
    if hc.handler.databricks != nil {
        if err := hc.checkDatabricks(ctx); err != nil {
            status.Providers["databricks"] = fmt.Sprintf("unhealthy: %v", err)
            status.Healthy = false
        } else {
            status.Providers["databricks"] = "healthy"
        }
    }

    // Check I/O
    if hc.handler.io != nil {
        status.Providers["io"] = "healthy"
    }

    // Check Workspace
    if hc.handler.workspace != nil {
        status.Providers["workspace"] = "healthy"
    }

    return status
}

func (hc *HealthChecker) checkDatabricks(ctx context.Context) error {
    // Try to list catalogs as a basic health check
    _, err := hc.handler.databricks.ListCatalogs(ctx)
    return err
}
```

### 7.3 Implement Graceful Shutdown

**pkg/server/server.go:**

```go
type Server struct {
    handler   *CombinedHandler
    mcpServer *mcp.Server
    logger    *slog.Logger
    shutdown  chan os.Signal
}

func NewServer(handler *CombinedHandler, logger *slog.Logger) *Server {
    transport := mcp.NewStdioTransport()
    mcpServer := mcp.NewServer(handler, transport, logger)

    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    return &Server{
        handler:   handler,
        mcpServer: mcpServer,
        logger:    logger,
        shutdown:  shutdown,
    }
}

func (s *Server) Run(ctx context.Context) error {
    // Create cancellable context
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    // Start server in goroutine
    errChan := make(chan error, 1)
    go func() {
        s.logger.Info("MCP server starting")
        errChan <- s.mcpServer.Serve(ctx)
    }()

    // Wait for shutdown signal or error
    select {
    case <-s.shutdown:
        s.logger.Info("shutdown signal received")
        cancel()
        // Wait for server to stop (with timeout)
        select {
        case err := <-errChan:
            return err
        case <-time.After(5 * time.Second):
            return errors.New("shutdown timeout")
        }
    case err := <-errChan:
        return err
    }
}
```

### 7.4 Update Main Entry Point

**cmd/go-mcp/main.go:**

```go
func main() {
    // Parse flags
    var (
        disallowDeployment = flag.Bool("disallow-deployment", false, "Disable deployment operations")
        withWorkspace      = flag.Bool("with-workspace-tools", false, "Enable workspace tools")
        configPath         = flag.String("config", "", "Override config file path")
        showVersion        = flag.Bool("version", false, "Print version and exit")
        checkEnv           = flag.Bool("check", false, "Check environment and exit")
    )
    flag.Parse()

    if *showVersion {
        fmt.Printf("go-mcp version %s (commit: %s, built: %s)\n",
            version.Version, version.Commit, version.BuildTime)
        return
    }

    // Load config
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    if *configPath != "" {
        // Load from custom path
    }

    if err := cfg.LoadFromEnv(); err != nil {
        log.Fatalf("Failed to load env overrides: %v", err)
    }

    // Apply CLI overrides
    if *disallowDeployment {
        cfg.AllowDeployment = false
    }
    if *withWorkspace {
        cfg.WithWorkspaceTools = true
    }

    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    // Create session logger
    sessionID := fmt.Sprintf("%d-%s", time.Now().Unix(), randomString(8))
    logger := logging.NewLogger(sessionID, true)

    if *checkEnv {
        // Run environment check
        if err := checkEnvironment(cfg, logger); err != nil {
            os.Exit(1)
        }
        return
    }

    // Version check in background
    go func() {
        if err := version.CheckForUpdates(context.Background()); err != nil {
            logger.Debug("version check failed", "error", err)
        }
    }()

    // Print startup banner to stderr
    fmt.Fprintf(os.Stderr, "🚀 Go MCP Server v%s\n", version.Version)
    fmt.Fprintf(os.Stderr, "   Session ID: %s\n", sessionID)
    fmt.Fprintf(os.Stderr, "   Log file: %s\n\n", logging.GetLogPath(sessionID))

    // Create handler
    handler, err := server.NewCombinedHandler(cfg, logger)
    if err != nil {
        log.Fatalf("Failed to create handler: %v", err)
    }

    // Create and run server
    srv := server.NewServer(handler, logger)
    if err := srv.Run(context.Background()); err != nil {
        logger.Error("server error", "error", err)
        os.Exit(1)
    }

    logger.Info("server shutdown complete")
}

func checkEnvironment(cfg *config.Config, logger *slog.Logger) error {
    fmt.Println("🔍 Checking environment configuration...\n")

    checks := []struct {
        name  string
        check func() error
    }{
        {"Docker availability", checkDocker},
        {"Databricks connection", func() error {
            if cfg.DatabricksHost == "" {
                return errors.New("DATABRICKS_HOST not set")
            }
            return nil
        }},
    }

    allPassed := true
    for _, c := range checks {
        fmt.Printf("  %s... ", c.name)
        if err := c.check(); err != nil {
            fmt.Printf("✗\n    Error: %v\n", err)
            allPassed = false
        } else {
            fmt.Println("✓")
        }
    }

    if allPassed {
        fmt.Println("\n✅ All checks passed!")
        return nil
    } else {
        fmt.Println("\n❌ Some checks failed. Please review the errors above.")
        return errors.New("environment check failed")
    }
}

func checkDocker() error {
    cmd := exec.Command("docker", "ps")
    return cmd.Run()
}

func randomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[rand.Intn(len(charset))]
    }
    return string(b)
}
```

### 7.5 Create Integration Tests

**test/integration/full_workflow_test.go:**

```go
func TestFullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Create temp directory
    workDir := t.TempDir()

    // Create mock MCP client
    client := newMockMCPClient(t)

    // Test 1: Initialize server
    info, err := client.Initialize()
    require.NoError(t, err)
    assert.Equal(t, "go-mcp", info.Name)

    // Test 2: List tools
    tools, err := client.ListTools()
    require.NoError(t, err)
    assert.NotEmpty(t, tools)

    // Test 3: Scaffold project
    result, err := client.CallTool("scaffold_data_app", map[string]any{
        "work_dir": workDir,
    })
    require.NoError(t, err)
    assert.False(t, result.IsError)

    // Verify files created
    assert.FileExists(t, filepath.Join(workDir, "package.json"))

    // Test 4: Read file
    content, err := client.CallTool("read_file", map[string]any{
        "file_path": "package.json",
    })
    require.NoError(t, err)
    assert.Contains(t, content.Content[0].Text, "name")

    // Test 5: Write file
    _, err = client.CallTool("write_file", map[string]any{
        "file_path": "test.txt",
        "content":   "hello world",
    })
    require.NoError(t, err)

    // Test 6: Bash execution
    result, err = client.CallTool("bash", map[string]any{
        "command": "ls -la",
    })
    require.NoError(t, err)
    assert.Contains(t, result.Content[0].Text, "test.txt")

    // Test 7: Grep
    result, err = client.CallTool("grep", map[string]any{
        "pattern": "hello",
    })
    require.NoError(t, err)
    assert.Contains(t, result.Content[0].Text, "test.txt")
}

func TestDatabricksIntegration(t *testing.T) {
    if os.Getenv("DATABRICKS_HOST") == "" {
        t.Skip("Skipping Databricks integration test")
    }

    client := newMockMCPClient(t)

    // Test list catalogs
    result, err := client.CallTool("databricks_list_catalogs", nil)
    require.NoError(t, err)
    assert.NotEmpty(t, result.Content)

    // Test list schemas
    // Test list tables
    // Test describe table
    // Test execute query
}
```

### 7.6 Add Benchmarks

**test/benchmark/performance_test.go:**

```go
func BenchmarkServer_ListTools(b *testing.B)
func BenchmarkServer_CallTool(b *testing.B)
func BenchmarkSandbox_Operations(b *testing.B)
func BenchmarkDatabricks_ListCatalogs(b *testing.B)
```

### 7.7 Create Example Usage

**examples/basic_usage.go:**

```go
// Example demonstrating basic usage of the Go MCP server
func main() {
    // This would typically be called by an MCP client like Claude Code
}
```

## Acceptance Criteria

- [ ] Combined handler integrates all providers
- [ ] Health checks report provider status
- [ ] Graceful shutdown on SIGTERM/SIGINT
- [ ] Session management tracks work directory
- [ ] CLI flags work correctly
- [ ] Environment check validates setup
- [ ] Integration tests pass
- [ ] Server runs with real MCP client
- [ ] Error handling is robust
- [ ] Logging captures all operations

## Testing Commands

```bash
# Build
go build -o go-mcp ./cmd/go-mcp

# Run checks
./go-mcp --check

# Run server
./go-mcp

# Integration tests
go test ./test/integration/... -v

# Benchmarks
go test -bench=. ./test/benchmark/...

# Full test suite
go test ./... -cover
```

## Next Steps

Proceed to Step 8: CLI and Deployment once all acceptance criteria are met.
