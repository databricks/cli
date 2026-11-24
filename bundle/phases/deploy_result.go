package phases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// WriteDeployResultOnExit writes how the deploy ended to the file named by
// DATABRICKS_AUTO_DEPLOY_RESULT_FILE: {} on success, otherwise the failure's class. The caller defers
// it, so it also runs while a panic unwinds; it reports a panic as SERVICE, because a panic is a CLI
// bug rather than a problem with the bundle, and then re-panics. A missing file means the deploy
// never got far enough to write it. retErr points at the command's error.
func WriteDeployResultOnExit(ctx context.Context, retErr *error) {
	path, _ := env.AutoDeployResultFile(ctx)
	if r := recover(); r != nil {
		if path != "" {
			writeDeployResult(ctx, path, deployResult{ErrorClass: errorClassService})
		}
		panic(r)
	}
	if path == "" {
		return
	}

	var result deployResult
	switch {
	case logdiag.HasError(ctx):
		// The error deploy telemetry reports; later errors are often its consequences.
		result.ErrorClass = errorClass(logdiag.GetFirstErrorCause(ctx))
	case *retErr != nil:
		result.ErrorClass = errorClass(*retErr)
	}
	writeDeployResult(ctx, path, result)
}

func writeDeployResult(ctx context.Context, path string, result deployResult) {
	data, err := json.Marshal(result)
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	if err != nil {
		log.Warnf(ctx, "Failed to write the deploy result to %s: %s", filepath.ToSlash(path), err)
	}
}

// errorClass reports SERVICE when the platform answered with a failure: an API error with status
// 5xx, 429 or 408. Anything else is USER, network errors included, since a wrong host, proxy or
// certificate causes those too. When in doubt it says USER, because the runner shows the customer
// the message of a USER failure and hides a SERVICE one.
func errorClass(err error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return errorClassUser
	}
	code := apiErr.StatusCode
	if code >= http.StatusInternalServerError || code == http.StatusTooManyRequests || code == http.StatusRequestTimeout {
		return errorClassService
	}
	return errorClassUser
}
