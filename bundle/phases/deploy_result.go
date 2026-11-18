package phases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

const (
	errorClassUser    = "USER"
	errorClassService = "SERVICE"
)

// deployResult is what a deploy reports in the result file. Fields that don't apply are left out.
type deployResult struct {
	// ErrorClass is set only when the deploy failed.
	ErrorClass string `json:"error_class,omitempty"`
}

// WriteDeployResult writes how the deploy ended to the file named by
// DATABRICKS_AUTO_DEPLOY_RESULT_FILE: {} on success, otherwise the failure's class. The caller runs
// it on every exit path, so a missing file means the CLI crashed. retErr is the command's error.
func WriteDeployResult(ctx context.Context, retErr error) {
	path, _ := env.AutoDeployResultFile(ctx)
	if path == "" {
		return
	}

	var result deployResult
	switch {
	case logdiag.HasError(ctx):
		// The error deploy telemetry reports; later errors are often its consequences.
		result.ErrorClass = errorClass(logdiag.GetFirstErrorCause(ctx))
	case retErr != nil:
		result.ErrorClass = errorClass(retErr)
	}

	data, err := json.Marshal(result)
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	if err != nil {
		log.Warnf(ctx, "Failed to write the deploy result to %s: %s", filepath.ToSlash(path), err)
	}
}

// errorClass reports SERVICE when the platform failed: an API error with status 5xx, 429 or 408, or
// a request that got no response. Anything else is USER, including an error it can't inspect,
// because the runner shows the customer the message of a USER failure and hides a SERVICE one.
func errorClass(err error) string {
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		code := apiErr.StatusCode
		if code >= http.StatusInternalServerError || code == http.StatusTooManyRequests || code == http.StatusRequestTimeout {
			return errorClassService
		}
		return errorClassUser
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return errorClassService
	}
	return errorClassUser
}
