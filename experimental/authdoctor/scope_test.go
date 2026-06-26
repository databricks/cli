package authdoctor

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestClassifyScopeError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantKind   ScopeErrorKind
		wantScopes string
		wantOK     bool
	}{
		{
			name:       "access_denied scopes not assigned",
			err:        &oauth2.RetrieveError{ErrorCode: "access_denied", ErrorDescription: "Scopes 'jobs' are not assigned to the client abc-123"},
			wantKind:   ScopeErrorNotAssigned,
			wantScopes: "jobs",
			wantOK:     true,
		},
		{
			name:       "invalid_scope",
			err:        &oauth2.RetrieveError{ErrorCode: "invalid_scope", ErrorDescription: "Invalid scope: bogus"},
			wantKind:   ScopeErrorInvalid,
			wantScopes: "bogus",
			wantOK:     true,
		},
		{
			name:       "route level missing scopes 403",
			err:        &apierr.APIError{StatusCode: 403, Message: "Provided OAuth token does not have required scopes: sql"},
			wantKind:   ScopeErrorMissingAtRoute,
			wantScopes: "sql",
			wantOK:     true,
		},
		{
			name:       "wrapped oauth error",
			err:        fmt.Errorf("login: %w", &oauth2.RetrieveError{ErrorCode: "access_denied", ErrorDescription: "Scopes 'iam.users:read' are not assigned to the client x"}),
			wantKind:   ScopeErrorNotAssigned,
			wantScopes: "iam.users:read",
			wantOK:     true,
		},
		{
			name:   "access_denied without scope phrase is not a scope error",
			err:    &oauth2.RetrieveError{ErrorCode: "access_denied", ErrorDescription: "user is not allowed"},
			wantOK: false,
		},
		{
			name:   "plain 401 is not a scope error",
			err:    &apierr.APIError{StatusCode: 401, Message: "invalid token"},
			wantOK: false,
		},
		{
			name:   "nil",
			err:    nil,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, scopes, ok := ClassifyScopeError(tt.err)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantKind, kind)
				assert.Equal(t, tt.wantScopes, scopes)
			}
		})
	}
}

func TestCheckConnectivity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := healthySnapshot()
		s.ProbeAttempted = true
		s.ProbeUser = "alice@databricks.test"
		r := Analyze(s)
		f := findOne(t, r, checkNameConnectivity)
		assert.Equal(t, LevelOK, f.Level)
		assert.Contains(t, f.Title, "alice@databricks.test")
	})

	t.Run("scope error produces fix", func(t *testing.T) {
		s := healthySnapshot()
		s.ConfiguredScopes = []string{"sql"}
		s.ProbeAttempted = true
		s.ProbeErr = &oauth2.RetrieveError{ErrorCode: "access_denied", ErrorDescription: "Scopes 'jobs' are not assigned to the client abc"}
		r := Analyze(s)
		f := findOne(t, r, checkNameConnectivity)
		assert.Equal(t, LevelError, f.Level)
		// Fix should request both the configured and the missing scope.
		assert.Contains(t, f.Fix, "sql")
		assert.Contains(t, f.Fix, "jobs")
	})

	t.Run("invalid scope is not re-requested in the fix", func(t *testing.T) {
		s := healthySnapshot()
		s.ConfiguredScopes = []string{"sql"}
		s.ProbeAttempted = true
		s.ProbeErr = &oauth2.RetrieveError{ErrorCode: "invalid_scope", ErrorDescription: "Invalid scope: bogus"}
		r := Analyze(s)
		f := findOne(t, r, checkNameConnectivity)
		assert.Equal(t, LevelError, f.Level)
		assert.Contains(t, f.Fix, "sql")
		assert.NotContains(t, f.Fix, "bogus")
	})

	t.Run("generic error", func(t *testing.T) {
		s := healthySnapshot()
		s.ProbeAttempted = true
		s.ProbeErr = errors.New("connection refused")
		r := Analyze(s)
		f := findOne(t, r, checkNameConnectivity)
		assert.Equal(t, LevelError, f.Level)
		assert.Contains(t, f.Detail, "connection refused")
	})
}

func TestNeededScopes(t *testing.T) {
	assert.Equal(t, []string{"all-apis"}, neededScopes("", nil))
	assert.Equal(t, []string{"sql", "jobs"}, neededScopes("jobs", []string{"sql"}))
	// Dedup keeps the configured order then appends new ones.
	assert.Equal(t, []string{"sql", "jobs"}, neededScopes("sql, jobs", []string{"sql"}))
}

func TestScopeFixCommand(t *testing.T) {
	assert.Equal(t, "databricks auth login --profile P --scopes 'jobs,sql'", scopeFixCommand("P", []string{"jobs", "sql"}))
	assert.Equal(t, "databricks auth login --scopes 'all-apis'", scopeFixCommand("", nil))
	// A profile name with a space is shell-quoted so the command stays safe.
	assert.Equal(t, "databricks auth login --profile 'my profile' --scopes 'all-apis'", scopeFixCommand("my profile", []string{"all-apis"}))
}

func TestLoginFixQuotesProfile(t *testing.T) {
	assert.Equal(t, "databricks auth login", loginFix(""))
	assert.Equal(t, "databricks auth login --profile P", loginFix("P"))
	assert.Equal(t, "databricks auth login --profile 'my profile'", loginFix("my profile"))
}

func TestContainsScope(t *testing.T) {
	assert.True(t, containsScope([]string{"All-Apis"}, "all-apis"))
	assert.False(t, containsScope([]string{"jobs"}, "all-apis"))
}
