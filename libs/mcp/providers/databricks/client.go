package databricks

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/databricks-sdk-go"
)

// Client represents a Databricks client
type Client struct {
	workspace *databricks.WorkspaceClient
	config    *mcp.Config
}

// NewClient creates a new Databricks client
func NewClient(cfg *mcp.Config, ctx context.Context) (*Client, error) {
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
	}, nil
}
