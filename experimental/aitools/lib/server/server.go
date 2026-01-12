// Package server provides the main MCP server implementation with provider registration and lifecycle management.
package server

import (
	"context"

	mcp "github.com/databricks/cli/experimental/aitools/lib"
	mcpsdk "github.com/databricks/cli/experimental/aitools/lib/mcp"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/providers/clitools"
	"github.com/databricks/cli/experimental/aitools/lib/providers/sdkdocs"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/experimental/aitools/lib/trajectory"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
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
		Name:    "databricks-aitools",
		Version: build.GetInfo().Version,
	}

	sess := session.NewSession()
	server := mcpsdk.NewServer(impl, nil, sess)

	tracker, err := trajectory.NewTracker(ctx, sess, cfg)
	if err != nil {
		log.Warnf(ctx, "failed to create trajectory tracker: %v", err)
		tracker = nil
	}

	server.AddMiddleware(middlewares.NewToolCounterMiddleware(sess))
	server.AddMiddleware(middlewares.NewDatabricksClientMiddleware([]string{"databricks_configure_auth"}))
	server.AddMiddleware(middlewares.NewTrajectoryMiddleware(tracker))

	sess.SetTracker(tracker)

	// Set enabled capabilities for this MCP server
	sess.Set(session.CapabilitiesDataKey, []string{"apps"})

	return &Server{
		server:  server,
		config:  cfg,
		session: sess,
		tracker: tracker,
	}
}

// RegisterTools registers all configured providers and their tools with the server.
// CLItools provider is always registered.
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

	// Always register clitools provider
	if err := s.registerCLIToolsProvider(ctx); err != nil {
		return err
	}

	// Register SDK docs provider
	if err := s.registerSDKDocsProvider(ctx); err != nil {
		return err
	}

	return nil
}

// registerCLIToolsProvider registers the CLI tools provider
func (s *Server) registerCLIToolsProvider(ctx context.Context) error {
	log.Info(ctx, "Registering CLI tools provider")

	provider, err := clitools.NewProvider(ctx, s.config, s.session)
	if err != nil {
		return err
	}

	if err := provider.RegisterTools(s.server); err != nil {
		return err
	}

	return nil
}

// registerSDKDocsProvider registers the SDK documentation provider
func (s *Server) registerSDKDocsProvider(ctx context.Context) error {
	log.Info(ctx, "Registering SDK docs provider")

	provider, err := sdkdocs.NewProvider(ctx, s.config, s.session)
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
	client, err := databricks.NewWorkspaceClient()
	if err != nil {
		return err
	}

	// Verify authentication by getting current user
	me, err := client.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	// Store client in session for reuse
	s.session.Set(middlewares.DatabricksClientKey, client)

	// Log authenticated user
	if me.UserName != "" {
		log.Infof(ctx, "Authenticated with Databricks as: %s", me.UserName)
	}

	return nil
}
