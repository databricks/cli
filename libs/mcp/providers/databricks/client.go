package databricks

import (
	"fmt"
	"log/slog"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/databricks-sdk-go"
)

// Client represents a Databricks client
type Client struct {
	workspace *databricks.WorkspaceClient
	config    *mcp.Config
	logger    *slog.Logger
}

// NewClient creates a new Databricks client
func NewClient(cfg *mcp.Config, logger *slog.Logger) (*Client, error) {
	var workspace *databricks.WorkspaceClient
	var err error

	if cfg.DatabricksHost != "" {
		dbConfig := &databricks.Config{
			Host: cfg.DatabricksHost,
		}
		workspace, err = databricks.NewWorkspaceClient(dbConfig)
	} else {
		workspace, err = databricks.NewWorkspaceClient()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Databricks client: %w", err)
	}

	return &Client{
		workspace: workspace,
		config:    cfg,
		logger:    logger,
	}, nil
}
