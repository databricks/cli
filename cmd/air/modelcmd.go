package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"unicode/utf16"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/spf13/cobra"
)

// This file holds what `air register` and `air serve` share: their JSON error codes,
// how they report a wait they stopped, and the names and numbers both files take.

// Errors a model command wraps when it stops waiting, as "air register was
// interrupted". The server keeps working, and the error says how to continue.
var (
	errInterrupted = errors.New("was interrupted")
	errTimedOut    = errors.New("timed out")
)

// JSON error codes of the model commands' own failures; API errors keep the server's.
const (
	codeInvalidArgs   = "INVALID_ARGS"
	codeInterrupted   = "INTERRUPTED"
	codeTimeout       = "TIMEOUT"
	codeStagingFailed = "STAGING_FAILED"
	// codeEndpointNotReady: the endpoint serves the version but isn't ready, which
	// running air serve again doesn't change.
	codeEndpointNotReady = "ENDPOINT_NOT_READY"
	codeInternalError    = "INTERNAL_ERROR"
)

// apiErrorCodeRe matches an error code the server sent. The SDK fills ErrorCode
// with "" or "<nil>" when the response body has none.
var apiErrorCodeRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Z][A-Z_]*$`) })

// modelError carries the JSON error code and retryability of a failure whose
// message is err's.
type modelError struct {
	code      string
	retryable bool
	err       error
}

func (e *modelError) Error() string { return e.err.Error() }

func (e *modelError) Unwrap() error { return e.err }

func withCode(code string, retryable bool, err error) error {
	return &modelError{code: code, retryable: retryable, err: err}
}

// stoppedWaiting returns why a model command stopped waiting, or nil if it didn't.
// The SDK's endpoint poller reports a canceled context as its own timeout, so the
// context decides between Ctrl-C and --timeout.
func stoppedWaiting(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errInterrupted
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errTimedOut
	}
	if _, ok := errors.AsType[*retries.ErrTimedOut](err); ok {
		return errTimedOut
	}
	return nil
}

// modelErrorCode classifies a failed model command for the JSON error envelope.
// Failures that aren't the user's input or the server's answer, such as a lost
// connection or a failed endpoint update, are worth retrying with the printed
// command.
func modelErrorCode(err error) (code string, retryable bool) {
	switch {
	case errors.Is(err, errInterrupted):
		return codeInterrupted, true
	case errors.Is(err, errTimedOut):
		return codeTimeout, true
	}
	if coded, ok := errors.AsType[*modelError](err); ok {
		return coded.code, coded.retryable
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		// The statuses libs/retry repeats, plus 429, which the SDK repeats.
		retryable = apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode == http.StatusRequestTimeout || apiErr.StatusCode >= http.StatusInternalServerError
		if !apiErrorCodeRe().MatchString(apiErr.ErrorCode) {
			return codeInternalError, retryable
		}
		return apiErr.ErrorCode, retryable
	}
	return codeInternalError, true
}

// serveCommandPath is the command air register's output points to for serving a
// version, and whose -h documents serveConfig.
const serveCommandPath = "databricks air serve"

// Flags the model commands share.
const (
	ucModelFlag      = "uc-model"
	endpointNameFlag = "endpoint-name"
	modelVersionFlag = "model-version"
	modelSourceFlag  = "model-source"
)

// changedFlags reports which of names were passed on the command line.
func changedFlags(cmd *cobra.Command, names ...string) map[string]bool {
	changed := map[string]bool{}
	for _, name := range names {
		changed[name] = cmd.Flags().Changed(name)
	}
	return changed
}

// renderModelError renders a model command's failure, if any, with its JSON error
// code and whether retrying can help.
func renderModelError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	code, retryable := modelErrorCode(err)
	kind := "PERMANENT"
	if retryable {
		kind = "TRANSIENT"
	}
	return renderError(cmd.Context(), cmd, code, kind, retryable, err)
}

// schemaHelp is the help of a model command whose YAML schema documents a field
// given as a positional argument to -h, as in `-h config.compute`. cobra passes
// -h's positional args to the help func before Args/required-flag validation, so
// a field is documented without needing -f.
func schemaHelp(schema func() configField, commandPath string) func(*cobra.Command, []string) {
	return func(c *cobra.Command, args []string) {
		fields := c.Flags().Args()
		if len(fields) == 0 {
			// Parent() is nil for a detached command (unit tests).
			if parent := c.Parent(); parent != nil {
				parent.HelpFunc()(c, args)
				return
			}
			_ = c.Usage()
			return
		}
		field, err := resolveSchemaField(schema(), fields[0])
		if err != nil {
			c.PrintErrln("Error:", err)
			return
		}
		renderConfigField(c.OutOrStdout(), commandPath, field, nil)
	}
}

// registeredModelRe matches <catalog>.<schema>.<model> with the characters the
// server accepts in each part.
var registeredModelRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile("^[^\\s./`]+\\.[^\\s./`]+\\.[^\\s./`]+$") })

// modelVersionRe matches a Unity Catalog model version number.
var modelVersionRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[1-9][0-9]*$`) })

// validateUCModelName checks a Unity Catalog model name from source, the YAML field
// or a flag.
func validateUCModelName(name, source string) error {
	if !registeredModelRe().MatchString(name) {
		return fmt.Errorf("%s must be a Unity Catalog model name as <catalog>.<schema>.<model>, got %q", source, name)
	}
	return nil
}

// serverLength is the length of s as the server's limits count it, in UTF-16
// code units.
func serverLength(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}
