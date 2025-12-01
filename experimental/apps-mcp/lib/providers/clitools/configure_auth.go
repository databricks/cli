package clitools

import (
	"context"
	"errors"
	"os"

	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/databricks-sdk-go"
)

// ConfigureAuth creates and validates a Databricks workspace client with optional host and profile.
// The authenticated client is stored in the session data for reuse across tool calls.
func ConfigureAuth(ctx context.Context, sess *session.Session, host, profile *string) (*databricks.WorkspaceClient, error) {
	// Skip auth check if testing
	if os.Getenv("DATABRICKS_MCP_SKIP_AUTH_CHECK") == "1" {
		return nil, nil
	}

	cfg := &databricks.Config{}
	if host != nil || profile != nil {
		if host != nil {
			cfg.Host = *host
		}
		if profile != nil {
			cfg.Profile = *profile
		}
	}

	var client *databricks.WorkspaceClient
	client, err := databricks.NewWorkspaceClient(cfg)
	if err != nil {
		return nil, err
	}

	_, err = client.CurrentUser.Me(ctx)
	if err != nil {
		if profile == nil && host != nil {
			return nil, errors.New(prompts.MustExecuteTemplate("auth_u2m.tmpl", map[string]string{
				"WorkspaceURL": *host,
			}))
		}
		return nil, middlewares.WrapAuthError(ctx, err)
	}

	// Store client and profile in session data
	sess.Set(middlewares.DatabricksClientKey, client)
	if profile != nil {
		sess.Set(middlewares.DatabricksProfileKey, *profile)
	}

	return client, nil
}
