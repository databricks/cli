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

func TestFromErrErrorTemplate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "plain error has no template",
			err:  errors.New("resources.jobs.my_job failed"),
			want: "",
		},
		{
			name: "safeerr error",
			err:  safeerr.Errorf("cannot update %s: %w", "resources.jobs.my_job", errors.New("boom")),
			want: "cannot update %s: %w",
		},
		{
			name: "api error without any safeerr wrapping",
			err:  apiError("PERMISSION_DENIED", http.StatusForbidden),
			want: "403 PERMISSION_DENIED",
		},
		{
			name: "safeerr error wrapping an api error",
			err:  safeerr.Errorf("cannot update %s: %w", "resources.jobs.my_job", apiError("PERMISSION_DENIED", http.StatusForbidden)),
			want: "cannot update %s: %w [403 PERMISSION_DENIED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := FromErr(tt.err)
			assert.Len(t, diags, 1)
			assert.Equal(t, tt.want, diags[0].ErrorTemplate)
		})
	}
}

// TestFromErrErrorTemplateCarriesNoUserData is the property the field exists
// for: whatever the summary holds, the template holds none of it.
func TestFromErrErrorTemplateCarriesNoUserData(t *testing.T) {
	err := safeerr.Errorf("cannot update %s: %w",
		"resources.jobs.my_secret_job",
		apiError("PERMISSION_DENIED", http.StatusForbidden))

	diags := FromErr(err)
	assert.Len(t, diags, 1)

	for _, secret := range []string{"my_secret_job", "alice@example.com", "/Workspace/"} {
		assert.Contains(t, diags[0].Summary, secret, "summary carries user data, as before")
		assert.NotContains(t, diags[0].ErrorTemplate, secret, "template must not")
	}
}

func TestFromErrNil(t *testing.T) {
	assert.Nil(t, FromErr(nil))
}

// TestErrorTemplateMatchesFromErr keeps the exported helper and the field in
// step, since callers holding an error use one and callers holding a diagnostic
// use the other.
func TestErrorTemplateMatchesFromErr(t *testing.T) {
	for _, err := range []error{
		errors.New("plain"),
		safeerr.Errorf("cannot update %s: %w", "resources.jobs.my_job", errors.New("boom")),
		apiError("PERMISSION_DENIED", http.StatusForbidden),
		safeerr.Errorf("x: %w", apiError("ABORTED", http.StatusConflict)),
	} {
		assert.Equal(t, FromErr(err)[0].ErrorTemplate, ErrorTemplate(err))
	}
}
