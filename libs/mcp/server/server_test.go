package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/appdotbuild/go-mcp/pkg/config"
	"github.com/appdotbuild/go-mcp/pkg/logging"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServer_Initialize(t *testing.T) {
	logger, err := logging.NewLogger("test", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}

	server := NewServer(cfg, logger)
	if err := server.RegisterTools(); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)

	t1, t2 := mcp.NewInMemoryTransports()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.server.Run(serverCtx, t1)
	}()

	ctx := context.Background()
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	initResult := session.InitializeResult()
	if initResult == nil {
		t.Fatalf("InitializeResult is nil")
	}

	if initResult.ServerInfo.Name != "go-mcp" {
		t.Errorf("Expected server name 'go-mcp', got %s", initResult.ServerInfo.Name)
	}
}

func TestNewServer(t *testing.T) {
	logger, err := logging.NewLogger("test-newsrv", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.server == nil {
		t.Error("server.server is nil")
	}

	if server.logger == nil {
		t.Error("server.logger is nil")
	}

	if server.config == nil {
		t.Error("server.config is nil")
	}

	if server.session == nil {
		t.Error("server.session is nil")
	}
}

func TestRegisterTools_WithWorkspaceTools(t *testing.T) {
	logger, err := logging.NewLogger("test-workspace", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		WithWorkspaceTools: true,
	}

	server := NewServer(cfg, logger)
	if err := server.RegisterTools(); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
}

func TestRegisterTools_WithDeployment(t *testing.T) {
	if os.Getenv("DATABRICKS_HOST") == "" || os.Getenv("DATABRICKS_TOKEN") == "" {
		t.Skip("Skipping test: DATABRICKS_HOST and DATABRICKS_TOKEN required")
	}

	logger, err := logging.NewLogger("test-deployment", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		AllowDeployment: true,
	}

	server := NewServer(cfg, logger)
	if err := server.RegisterTools(); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
}

func TestRegisterTools_AllProviders(t *testing.T) {
	if os.Getenv("DATABRICKS_HOST") == "" || os.Getenv("DATABRICKS_TOKEN") == "" {
		t.Skip("Skipping test: DATABRICKS_HOST and DATABRICKS_TOKEN required")
	}

	logger, err := logging.NewLogger("test-all", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		WithWorkspaceTools: true,
		AllowDeployment:    true,
	}

	server := NewServer(cfg, logger)
	if err := server.RegisterTools(); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}
}

func TestGetServer(t *testing.T) {
	logger, err := logging.NewLogger("test-getserver", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	mcpServer := server.GetServer()
	if mcpServer == nil {
		t.Error("GetServer returned nil")
	}

	if mcpServer != server.server {
		t.Error("GetServer did not return the internal server")
	}
}

func TestGetTracker(t *testing.T) {
	logger, err := logging.NewLogger("test-gettracker", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	tracker := server.GetTracker()
	if tracker != server.tracker {
		t.Error("GetTracker did not return the internal tracker")
	}
}

func TestShutdown(t *testing.T) {
	logger, err := logging.NewLogger("test-shutdown", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestShutdown_WithTracker(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	logger, err := logging.NewLogger("test-shutdown-tracker", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	if server.tracker == nil {
		t.Skip("Tracker is nil, skipping test")
	}

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestCheckHealth_Minimal(t *testing.T) {
	logger, err := logging.NewLogger("test-health", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	ctx := context.Background()
	status := server.CheckHealth(ctx)

	if status == nil {
		t.Fatal("CheckHealth returned nil")
	}

	if !status.Healthy {
		t.Error("Server should be healthy")
	}

	if status.Providers == nil {
		t.Fatal("Providers map is nil")
	}

	if status.Providers["databricks"] != "healthy" {
		t.Errorf("databricks provider should be healthy, got: %s", status.Providers["databricks"])
	}

	if status.Providers["io"] != "healthy" {
		t.Errorf("io provider should be healthy, got: %s", status.Providers["io"])
	}

	if status.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestCheckHealth_WithWorkspace(t *testing.T) {
	logger, err := logging.NewLogger("test-health-ws", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		WithWorkspaceTools: true,
	}
	server := NewServer(cfg, logger)

	ctx := context.Background()
	status := server.CheckHealth(ctx)

	if status.Providers["workspace"] != "healthy" {
		t.Errorf("workspace provider should be healthy, got: %s", status.Providers["workspace"])
	}
}

func TestCheckHealth_WithDeployment(t *testing.T) {
	logger, err := logging.NewLogger("test-health-deploy", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{
		AllowDeployment: true,
	}
	server := NewServer(cfg, logger)

	ctx := context.Background()
	status := server.CheckHealth(ctx)

	if status.Providers["deployment"] != "healthy" {
		t.Errorf("deployment provider should be healthy, got: %s", status.Providers["deployment"])
	}
}

func TestCheckHealth_CancelledContext(t *testing.T) {
	logger, err := logging.NewLogger("test-health-cancel", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := server.CheckHealth(ctx)
	if status == nil {
		t.Fatal("CheckHealth should not return nil even with cancelled context")
	}
}

func TestNewServer_WithTrajectoryFailure(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	invalidDir := filepath.Join(tempDir, "invalid")
	if err := os.WriteFile(invalidDir, []byte("file"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	os.Setenv("HOME", invalidDir)
	defer os.Setenv("HOME", originalHome)

	logger, err := logging.NewLogger("test-trajectory-fail", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := &config.Config{}
	server := NewServer(cfg, logger)

	if server == nil {
		t.Fatal("NewServer should not return nil even if trajectory fails")
	}
}
