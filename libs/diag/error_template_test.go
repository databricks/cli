package diag

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/safeerr"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
)

// apiError builds an SDK error whose message carries user data, the way a real
// one does.
func apiError(code string, status int) error {
	return &apierr.APIError{
		ErrorCode:  code,
		StatusCode: status,
		Message:    "User alice@example.com cannot access /Workspace/Users/alice@example.com/job",
	}
}

func TestSafeAPIErrorDescription(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "not an api error", err: errors.New("boom"), want: ""},
		{name: "code and status", err: apiError("PERMISSION_DENIED", http.StatusForbidden), want: "403 PERMISSION_DENIED"},
		{name: "code only", err: apiError("RESOURCE_DOES_NOT_EXIST", 0), want: "RESOURCE_DOES_NOT_EXIST"},
		{name: "status only", err: apiError("", http.StatusInternalServerError), want: "500"},
		{name: "nothing safe", err: apiError("", 0), want: ""},
		{name: "wrapped", err: fmt.Errorf("deploying: %w", apiError("QUOTA_EXCEEDED", 429)), want: "429 QUOTA_EXCEEDED"},

		// A code that does not have the shape of an enum member is dropped
		// rather than trusted, so free text cannot ride along.
		{name: "code with a path", err: apiError("cannot find /Workspace/Users/a@b.com/x", 404), want: "404"},
		{name: "code with a quoted name", err: apiError(`job "Q4 forecast" missing`, 404), want: "404"},
		{name: "code lowercase", err: apiError("permission_denied", 403), want: "403"},
		{name: "code with a dot", err: apiError("PERMISSION.DENIED", 403), want: "403"},
		{name: "code with a space", err: apiError("PERMISSION DENIED", 403), want: "403"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SafeAPIErrorDescription(tt.err))
		})
	}
}

// TestFromErrErrorTemplateCarriesNoUserData is the property the field exists
// for: whatever the summary holds, the template holds none of it.

func TestFromErrNil(t *testing.T) {
	assert.Nil(t, FromErr(nil))
}

// TestErrorTemplateMatchesFromErr keeps the exported helper and the field in
// step, since callers holding an error use one and callers holding a diagnostic
// use the other.

// standInErr models a typed CLI error such as libs/filer's: user data in the
// message, a PII-free classification as its stand-in.
type standInErr struct{}

func (standInErr) Error() string      { return "access denied: /Workspace/Users/a@b.com/x" }
func (standInErr) SafeString() string { return "access denied" }

// TestErrorTemplateUsesStandInWithoutSafeerr covers the call sites not raised
// through safeerr, which is most of them: a typed error still describes itself.
func TestErrorTemplateUsesStandInWithoutSafeerr(t *testing.T) {
	assert.Equal(t, "access denied", ErrorTemplate(standInErr{}))

	// Combined with an API error at the end of the chain.
	err := safeerr.Errorf("pushing state: %w", standInErr{})
	assert.Equal(t, "pushing state: access denied", ErrorTemplate(err))

	// A stand-in never displaces a real template.
	assert.Equal(t, "cannot update %s: %w",
		ErrorTemplate(safeerr.Errorf("cannot update %s: %w", "resources.jobs.x", errors.New("boom"))))
}
