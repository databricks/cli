// Package server provides the main MCP server implementation with provider registration and lifecycle management.
package server

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/providers/databricks"
	"github.com/databricks/cli/libs/mcp/providers/deployment"
	"github.com/databricks/cli/libs/mcp/providers/io"
	"github.com/databricks/cli/libs/mcp/providers/workspace"
	"github.com/databricks/cli/libs/mcp/session"
	"github.com/databricks/cli/libs/mcp/trajectory"
	"github.com/databricks/cli/libs/mcp/version"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server manages the MCP server lifecycle, provider registration, and session tracking.
type Server struct {
	server  *mcpsdk.Server
	config  *mcp.Config
	session *session.Session
	tracker *trajectory.Tracker
}

// NewServer creates and initializes a new MCP server instance.
// It creates a session, trajectory tracker, and prepares the server for provider registration.
func NewServer(cfg *mcp.Config, ctx context.Context) *Server {
	impl := &mcpsdk.Implementation{
		Name:    "go-mcp",
		Version: version.GetVersion(),
	}

	server := mcpsdk.NewServer(impl, nil)
	sess := session.NewSession()

	tracker, err := trajectory.NewTracker(sess, cfg, ctx)
	if err != nil {
		log.Warnf(ctx, "failed to create trajectory tracker: %v", err)
		tracker = nil
	}

	sess.Tracker = tracker

	return &Server{
		server:  server,
		config:  cfg,
		session: sess,
		tracker: tracker,
	}
}

// RegisterTools registers all configured providers and their tools with the server.
// Databricks and IO providers are always registered, while workspace and deployment
// providers are conditional based on configuration flags.
func (s *Server) RegisterTools(ctx context.Context) error {
	log.Infof(ctx, "Registering tools")

	// Always register databricks provider
	if err := s.registerDatabricksProvider(ctx); err != nil {
		return err
	}

	// Always register io provider
	if err := s.registerIOProvider(ctx); err != nil {
		return err
	}

	// Register workspace provider if enabled
	if s.config.WithWorkspaceTools {
		if err := s.registerWorkspaceProvider(ctx); err != nil {
			return err
		}
	}

	// Register deployment provider if enabled
	if s.config.AllowDeployment {
		log.Infof(ctx, "Deployment provider enabled")
		if err := s.registerDeploymentProvider(ctx); err != nil {
			return err
		}
	} else {
		log.Infof(ctx, "Deployment provider disabled (enable with allow_deployment: true)")
	}

	return nil
}

// registerDatabricksProvider registers the Databricks provider
func (s *Server) registerDatabricksProvider(ctx context.Context) error {
	log.Infof(ctx, "Registering Databricks provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := databricks.NewProvider(s.config, s.session, ctx)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerIOProvider registers the I/O provider
func (s *Server) registerIOProvider(ctx context.Context) error {
	log.Infof(ctx, "Registering I/O provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := io.NewProvider(s.config.IoConfig, s.session, ctx)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerWorkspaceProvider registers the workspace provider
func (s *Server) registerWorkspaceProvider(ctx context.Context) error {
	log.Infof(ctx, "Registering workspace provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := workspace.NewProvider(s.session, ctx)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerDeploymentProvider registers the deployment provider
func (s *Server) registerDeploymentProvider(ctx context.Context) error {
	log.Infof(ctx, "Registering deployment provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := deployment.NewProvider(s.config, s.session, ctx)
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
	log.Infof(ctx, "Starting MCP server with STDIO transport")

	transport := &mcpsdk.StdioTransport{}
	if err := s.server.Run(ctx, transport); err != nil {
		log.Errorf(ctx, "Server failed", "error", err)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server, closing the trajectory tracker and releasing resources.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Infof(ctx, "Shutting down MCP server")

	if s.tracker != nil {
		if err := s.tracker.Close(); err != nil {
			log.Warnf(ctx, "failed to close trajectory tracker", "error", err)
		}
	}

	return nil
}

// GetServer returns the underlying MCP SDK server instance for testing purposes.
func (s *Server) GetServer() *mcpsdk.Server {
	return s.server
}

// GetTracker returns the trajectory tracker used for recording tool calls.
// Providers use this to wrap their tool handlers for automatic trajectory logging.
func (s *Server) GetTracker() *trajectory.Tracker {
	return s.tracker
}
