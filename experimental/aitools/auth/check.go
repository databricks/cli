package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var (
	authCheckOnce   sync.Once
	authCheckResult error
)

// CheckAuthentication checks if the user is authenticated to a Databricks workspace.
// It caches the result so the check only runs once per process.
func CheckAuthentication(ctx context.Context) error {
	authCheckOnce.Do(func() {
		authCheckResult = checkAuth(ctx)
	})
	return authCheckResult
}

func checkAuth(ctx context.Context) error {
	if os.Getenv("DATABRICKS_MCP_SKIP_AUTH_CHECK") == "1" {
		return nil
	}

	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		return wrapAuthError(err)
	}

	// Use Jobs API for auth check (fast). Expected: 404 (authenticated), 401/403 (not authenticated).
	_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{JobId: 999999999})
	if err == nil {
		return nil
	}

	var apiErr *apierr.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusNotFound:
			return nil
		case http.StatusUnauthorized, http.StatusForbidden:
			return errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", nil))
		default:
			return nil
		}
	}

	return wrapAuthError(err)
}

func wrapAuthError(err error) error {
	if errors.Is(err, config.ErrCannotConfigureDefault) {
		return errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", nil))
	}
	return err
}
