package authdoctor

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go/apierr"
	"golang.org/x/oauth2"
)

// ScopeErrorKind classifies an OAuth token-downscoping failure. These map to
// the real, client-visible shapes (there is no "SCOPE_MISMATCH" error):
//   - access_denied at the token endpoint: requested scopes the client lacks.
//   - invalid_scope at the token endpoint: an unknown/malformed scope.
//   - HTTP 403 at the API route: the token lacks a scope the route requires.
//
// See go/downscoping and the auth-access-doctor design doc, resolved Q5.
type ScopeErrorKind string

const (
	ScopeErrorNotAssigned    ScopeErrorKind = "scopes_not_assigned"
	ScopeErrorInvalid        ScopeErrorKind = "invalid_scope"
	ScopeErrorMissingAtRoute ScopeErrorKind = "missing_required_scopes"
)

// OAuth2 error codes (RFC 6749 'error' parameter), as returned by the
// /oidc/v1/token endpoint.
const (
	oauthErrAccessDenied = "access_denied"
	oauthErrInvalidScope = "invalid_scope"
)

// scopeAllApis is the wildcard scope that removes any downscoping restriction.
const scopeAllApis = "all-apis"

// Documented substrings of the downscoping error descriptions. We branch on the
// structured ErrorCode / StatusCode first; these refine the classification and
// drive scope extraction for the fix command.
const (
	notAssignedPhrase    = "are not assigned to the client"
	invalidScopePhrase   = "invalid scope:"
	requiredScopesPhrase = "does not have required scopes:"
)

// ClassifyScopeError reports whether err is an OAuth scope/downscoping failure,
// its kind, and the scope list named in the error (best-effort, for display and
// the fix command). It branches on structured error fields, not on Error()
// text, except for the route-level 403 which has no structured scope code.
func ClassifyScopeError(err error) (ScopeErrorKind, string, bool) {
	if err == nil {
		return "", "", false
	}

	if re, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		desc := re.ErrorDescription
		switch re.ErrorCode {
		case oauthErrInvalidScope:
			return ScopeErrorInvalid, extractAfter(desc, invalidScopePhrase), true
		case oauthErrAccessDenied:
			if containsFold(desc, notAssignedPhrase) {
				return ScopeErrorNotAssigned, extractQuoted(desc), true
			}
		}
	}

	if ae, ok := errors.AsType[*apierr.APIError](err); ok {
		// The route-level scope error is a plain 403 with a documented message
		// and no structured scope code, so match the documented phrase.
		if ae.StatusCode == http.StatusForbidden && containsFold(ae.Message, requiredScopesPhrase) {
			return ScopeErrorMissingAtRoute, extractAfter(ae.Message, requiredScopesPhrase), true
		}
	}

	return "", "", false
}

func scopeErrorTitle(kind ScopeErrorKind) string {
	switch kind {
	case ScopeErrorInvalid:
		return "OAuth login requested an invalid scope"
	case ScopeErrorMissingAtRoute:
		return "OAuth token is missing a scope this command needs"
	default:
		return "OAuth token is missing required scopes"
	}
}

func scopeErrorDetail(kind ScopeErrorKind, scopes string, err error) string {
	switch kind {
	case ScopeErrorInvalid:
		return fmt.Sprintf("The authorization server rejected scope %q as invalid.", scopes)
	case ScopeErrorMissingAtRoute:
		return fmt.Sprintf("The request was rejected (403); the token lacks scope(s): %s.", emptyDash(scopes))
	default:
		return fmt.Sprintf("Scopes %q are not assigned to this OAuth client.", scopes)
	}
}

// scopeFixScopes picks the scopes to request in the remediation command for a
// scope error. An invalid scope must NOT be re-requested (it would fail the
// same way), so for that kind we suggest the configured scopes alone (or
// all-apis if none); for the other kinds we add the missing scope.
func scopeFixScopes(kind ScopeErrorKind, fromError string, configured []string) []string {
	if kind == ScopeErrorInvalid {
		if len(configured) == 0 {
			return []string{scopeAllApis}
		}
		return configured
	}
	return neededScopes(fromError, configured)
}

// neededScopes computes the scope set to request in the fix command: the union
// of the currently configured scopes and the scope(s) named in the error.
// Falls back to all-apis when nothing usable was parsed.
func neededScopes(fromError string, configured []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, s := range configured {
		add(s)
	}
	for _, s := range splitScopeList(fromError) {
		add(s)
	}
	if len(out) == 0 {
		return []string{scopeAllApis}
	}
	return out
}

// scopeFixCommand builds the auth login command that requests the given scopes.
// The profile name is shell-quoted so a name with spaces or metacharacters
// stays copy-paste-safe.
func scopeFixCommand(profile string, scopes []string) string {
	if len(scopes) == 0 {
		scopes = []string{scopeAllApis}
	}
	cmd := "databricks auth login"
	if profile != "" {
		cmd += " --profile " + shellquote.BashArg(profile)
	}
	return fmt.Sprintf("%s --scopes '%s'", cmd, strings.Join(scopes, ","))
}

func containsScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if strings.EqualFold(strings.TrimSpace(s), want) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// extractQuoted returns the content of the first single- or double-quoted run
// in s, or "" if none.
func extractQuoted(s string) string {
	for _, q := range []byte{'\'', '"'} {
		if i := strings.IndexByte(s, q); i >= 0 {
			if j := strings.IndexByte(s[i+1:], q); j >= 0 {
				return s[i+1 : i+1+j]
			}
		}
	}
	return ""
}

// extractAfter returns the text following the (case-insensitive) marker,
// trimmed, or "" if the marker is absent.
func extractAfter(s, marker string) string {
	idx := strings.Index(strings.ToLower(s), strings.ToLower(marker))
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(strings.Trim(s[idx+len(marker):], " '\"."))
}

// splitScopeList splits a scope list that may be comma- or space-separated.
func splitScopeList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	var out []string
	for _, f := range fields {
		f = strings.Trim(f, " '\".")
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
