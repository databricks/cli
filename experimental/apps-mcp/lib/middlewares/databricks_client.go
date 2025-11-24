package middlewares

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
)

const (
	DatabricksClientKey = "databricks_client"
)

func NewDatabricksClientMiddleware(unauthorizedToolNames []string) mcp.Middleware {
	return mcp.NewMiddleware(func(ctx *mcp.MiddlewareContext, next mcp.NextFunc) (*mcp.CallToolResult, error) {
		if slices.Contains(unauthorizedToolNames, ctx.Request.Tool.Name) {
			return next()
		}

		_, ok := ctx.Session.Get(DatabricksClientKey)
		if !ok {
			w, err := checkAuth(ctx.Ctx)
			if err != nil {
				return mcp.CreateNewTextContentResultError(err), nil
			}
			ctx.Session.Set(DatabricksClientKey, w)

			// Start background warehouse loading once client is initialized
			go loadWarehouseInBackground(ctx.Ctx)
		}

		return next()
	})
}

func MustGetApiClient(ctx context.Context) (*httpclient.ApiClient, error) {
	w := MustGetDatabricksClient(ctx)
	clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client config: %w", err)
	}
	return httpclient.NewApiClient(clientCfg), nil
}

func MustGetDatabricksClient(ctx context.Context) *databricks.WorkspaceClient {
	w, err := GetDatabricksClient(ctx)
	if err != nil {
		panic(err)
	}
	return w
}

func GetDatabricksClient(ctx context.Context) (*databricks.WorkspaceClient, error) {
	sess, err := session.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	w, ok := sess.Get(DatabricksClientKey)
	if !ok {
		return nil, errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", nil))
	}
	return w.(*databricks.WorkspaceClient), nil
}

func checkAuth(ctx context.Context) (*databricks.WorkspaceClient, error) {
	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		return nil, wrapAuthError(err)
	}

	_, err = w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, wrapAuthError(err)
	}

	return w, nil
}

func wrapAuthError(err error) error {
	if errors.Is(err, config.ErrCannotConfigureDefault) {
		return errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", nil))
	}
	return err
}
