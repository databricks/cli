// Package server provides the main MCP server implementation with provider registration and lifecycle management.
package server

import (
	"context"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/databricks"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/deployment"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/io"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/experimental/apps-mcp/lib/trajectory"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
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
func NewServer(ctx context.Context, cfg *mcp.Config) *Server {
	impl := &mcpsdk.Implementation{
		Name:    "databricks-apps-mcp",
		Version: build.GetInfo().Version,
	}

	server := mcpsdk.NewServer(impl, nil)
	sess := session.NewSession()

	tracker, err := trajectory.NewTracker(ctx, sess, cfg)
	if err != nil {
		log.Warnf(ctx, "failed to create trajectory tracker: %v", err)
		tracker = nil
	}

	server.AddMiddleware(middlewares.NewToolCounterMiddleware(sess))
	server.AddMiddleware(middlewares.NewDatabricksClientMiddleware([]string{"databricks_configure_auth"}))
	server.AddMiddleware(middlewares.NewEngineGuideMiddleware())
	server.AddMiddleware(middlewares.NewTrajectoryMiddleware(tracker))

	sess.SetTracker(tracker)

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
	log.Info(ctx, "Registering tools")

	// Add session to context for early initialization
	ctx = session.WithSession(ctx, s.session)

	// Eagerly initialize Databricks authentication if possible
	// This makes the first tool call faster by pre-authenticating
	if err := s.initializeDatabricksAuth(ctx); err != nil {
		log.Debugf(ctx, "Databricks authentication not initialized during startup: %v", err)
		// Don't fail - authentication will be attempted on first tool call via middleware
	}

	// Always register databricks provider
	if err := s.registerDatabricksProvider(ctx); err != nil {
		return err
	}

	// Always register io provider
	if err := s.registerIOProvider(ctx); err != nil {
		return err
	}

	// Register deployment provider if enabled
	if s.config.AllowDeployment {
		log.Info(ctx, "Deployment provider enabled")
		if err := s.registerDeploymentProvider(ctx); err != nil {
			return err
		}
	} else {
		log.Info(ctx, "Deployment provider disabled (enable with allow_deployment: true)")
	}

	return nil
}

// registerDatabricksProvider registers the Databricks provider
func (s *Server) registerDatabricksProvider(ctx context.Context) error {
	log.Info(ctx, "Registering Databricks provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := databricks.NewProvider(ctx, s.config, s.session)
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
	log.Info(ctx, "Registering I/O provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := io.NewProvider(ctx, s.config.IoConfig, s.session)
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
	log.Info(ctx, "Registering deployment provider")

	// Add session to context
	ctx = session.WithSession(ctx, s.session)

	provider, err := deployment.NewProvider(ctx, s.config, s.session)
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
	log.Info(ctx, "Starting MCP server with STDIO transport")

	transport := mcpsdk.NewStdioTransport()
	if err := s.server.Run(ctx, transport); err != nil {
		log.Errorf(ctx, "Server failed: error=%v", err)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server, closing the trajectory tracker and releasing resources.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info(ctx, "Shutting down MCP server")

	if s.tracker != nil {
		if err := s.tracker.Close(); err != nil {
			log.Warnf(ctx, "failed to close trajectory tracker: error=%v", err)
		}
	}

	return nil
}

// GetServer returns the underlying MCP SDK server instance for testing purposes.
func (s *Server) GetServer() *mcpsdk.Server {
	return s.server
}

// initializeDatabricksAuth attempts to eagerly authenticate with Databricks during startup.
// This improves the user experience by making the first tool call faster.
// If authentication fails, tools will still work via lazy authentication in the middleware.
func (s *Server) initializeDatabricksAuth(ctx context.Context) error {
	client, err := databricks.ConfigureAuth(ctx, s.session, nil, nil)
	if err != nil {
		return err
	}

	// Get current user info for logging
	if client != nil {
		me, err := client.CurrentUser.Me(ctx)
		if err == nil && me.UserName != "" {
			log.Infof(ctx, "Authenticated with Databricks as: %s", me.UserName)
		}
	}

	return nil
}
