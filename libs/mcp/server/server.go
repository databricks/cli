// Package mcp provides the main MCP server implementation with provider registration and lifecycle management.
package mcp

import (
	"context"
	"log/slog"

	"github.com/appdotbuild/go-mcp/pkg/config"
	"github.com/appdotbuild/go-mcp/pkg/providers/databricks"
	"github.com/appdotbuild/go-mcp/pkg/providers/deployment"
	"github.com/appdotbuild/go-mcp/pkg/providers/io"
	"github.com/appdotbuild/go-mcp/pkg/providers/workspace"
	"github.com/appdotbuild/go-mcp/pkg/session"
	"github.com/appdotbuild/go-mcp/pkg/trajectory"
	"github.com/appdotbuild/go-mcp/pkg/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server manages the MCP server lifecycle, provider registration, and session tracking.
type Server struct {
	server  *mcp.Server
	logger  *slog.Logger
	config  *config.Config
	session *session.Session
	tracker *trajectory.Tracker
}

// NewServer creates and initializes a new MCP server instance.
// It creates a session, trajectory tracker, and prepares the server for provider registration.
func NewServer(cfg *config.Config, logger *slog.Logger) *Server {
	impl := &mcp.Implementation{
		Name:    "go-mcp",
		Version: version.GetVersion(),
	}

	server := mcp.NewServer(impl, nil)
	sess := session.NewSession()

	tracker, err := trajectory.NewTracker(sess, cfg, logger)
	if err != nil {
		logger.Warn("failed to create trajectory tracker", "error", err)
		tracker = nil
	}

	sess.Tracker = tracker

	return &Server{
		server:  server,
		logger:  logger,
		config:  cfg,
		session: sess,
		tracker: tracker,
	}
}

// RegisterTools registers all configured providers and their tools with the server.
// Databricks and IO providers are always registered, while workspace and deployment
// providers are conditional based on configuration flags.
func (s *Server) RegisterTools() error {
	s.logger.Info("Registering tools")

	// Always register databricks provider
	if err := s.registerDatabricksProvider(); err != nil {
		return err
	}

	// Always register io provider
	if err := s.registerIOProvider(); err != nil {
		return err
	}

	// Register workspace provider if enabled
	if s.config.WithWorkspaceTools {
		if err := s.registerWorkspaceProvider(); err != nil {
			return err
		}
	}

	// Register deployment provider if enabled
	if s.config.AllowDeployment {
		s.logger.Info("Deployment provider enabled")
		if err := s.registerDeploymentProvider(); err != nil {
			return err
		}
	} else {
		s.logger.Info("Deployment provider disabled (enable with allow_deployment: true)")
	}

	return nil
}

// registerDatabricksProvider registers the Databricks provider
func (s *Server) registerDatabricksProvider() error {
	s.logger.Info("Registering Databricks provider")

	provider, err := databricks.NewProvider(s.config, s.session, s.logger)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerIOProvider registers the I/O provider
func (s *Server) registerIOProvider() error {
	s.logger.Info("Registering I/O provider")

	provider, err := io.NewProvider(s.config.IoConfig, s.session, s.logger)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerWorkspaceProvider registers the workspace provider
func (s *Server) registerWorkspaceProvider() error {
	s.logger.Info("Registering workspace provider")

	provider, err := workspace.NewProvider(s.session, s.logger)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerDeploymentProvider registers the deployment provider
func (s *Server) registerDeploymentProvider() error {
	s.logger.Info("Registering deployment provider")

	provider, err := deployment.NewProvider(s.config, s.session, s.logger)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// Run starts the MCP server with STDIO transport and blocks until the context is cancelled.
// The server communicates via standard input/output following the MCP protocol.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("Starting MCP server with STDIO transport")

	transport := &mcp.StdioTransport{}
	if err := s.server.Run(ctx, transport); err != nil {
		s.logger.Error("Server failed", "error", err)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server, closing the trajectory tracker and releasing resources.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down MCP server")

	if s.tracker != nil {
		if err := s.tracker.Close(); err != nil {
			s.logger.Warn("failed to close trajectory tracker", "error", err)
		}
	}

	return nil
}

// GetServer returns the underlying MCP SDK server instance for testing purposes.
func (s *Server) GetServer() *mcp.Server {
	return s.server
}

// GetTracker returns the trajectory tracker used for recording tool calls.
// Providers use this to wrap their tool handlers for automatic trajectory logging.
func (s *Server) GetTracker() *trajectory.Tracker {
	return s.tracker
}
