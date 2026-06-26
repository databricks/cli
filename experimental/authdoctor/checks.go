package authdoctor

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/cli/libs/shellquote"
)

// Check names, used as the Check field on findings and as stable keys for JSON
// consumers and tests.
const (
	checkNameResolution   = "resolution"
	checkNameHost         = "host"
	checkNameAuthType     = "auth_type"
	checkNameToken        = "token"
	checkNameScopes       = "scopes"
	checkNameClock        = "clock"
	checkNameConnectivity = "connectivity"
)

// clockSkewThreshold is how far the local clock may drift from the server before
// the doctor warns; OAuth/token validation typically tolerates a few minutes.
const clockSkewThreshold = 5 * time.Minute

// Host classifications. Exported so the cmd layer fills Snapshot.HostType with
// the same labels checkHost compares on.
const (
	HostTypeWorkspace = "workspace"
	HostTypeAccount   = "account"
	HostTypeUnified   = "unified"
)

// checkResolution reports the effective profile/host/auth type and flags the
// cases where the CLI's divergent resolution paths would pick different
// credentials from the same environment. This is the doctor's signature check.
func checkResolution(s Snapshot) []Finding {
	var out []Finding

	if s.Host == "" && s.Profile == "" {
		return []Finding{{
			Check:  checkNameResolution,
			Level:  LevelError,
			Title:  "No profile or host resolved",
			Detail: "Neither a profile nor a host could be resolved from flags, environment variables, or ~/.databrickscfg.",
			Fix:    "databricks auth login --host <workspace-url>",
		}}
	}

	profile := s.Profile
	if profile == "" {
		profile = "(none)"
	}
	out = append(out, Finding{
		Check:  checkNameResolution,
		Level:  LevelOK,
		Title:  fmt.Sprintf("Resolved profile %q from %s", profile, s.ProfileSource),
		Detail: fmt.Sprintf("host=%s, auth_type=%s", emptyDash(s.Host), emptyDash(s.AuthType)),
	})

	// Config that did not validate still resolves partial values above, but the
	// SDK would reject it, so surface the error rather than letting it look fine.
	if s.ConfigError != "" {
		out = append(out, Finding{
			Check:  checkNameResolution,
			Level:  LevelError,
			Title:  "Configuration did not resolve cleanly",
			Detail: s.ConfigError,
			Fix:    loginFix(s.Profile),
		})
	}

	// DATABRICKS_HOST overrides the profile's host (env attributes take
	// precedence over the profile in the SDK), while the profile still supplies
	// the token/auth type, so the token can be sent to the wrong host. --profile
	// does NOT undo this: the env attribute still wins, so the only fix is to
	// unset DATABRICKS_HOST. This env-first ordering also means `databricks auth
	// token` lands on the env host, so it diverges from the profile too.
	if s.EnvHost != "" && s.ProfileHost != "" && !sameHost(s.EnvHost, s.ProfileHost) {
		out = append(out, Finding{
			Check:  checkNameResolution,
			Level:  LevelWarn,
			Title:  "DATABRICKS_HOST overrides the profile host",
			Detail: fmt.Sprintf("Environment host %s shadows profile %q host %s; the profile's credentials are sent to the environment host.", s.EnvHost, s.Profile, s.ProfileHost),
			Fix:    "unset DATABRICKS_HOST (or set it to the profile host)",
		})
	}

	// DATABRICKS_TOKEN interacts with the profile differently depending on
	// whether the profile pins auth_type. With no explicit auth_type the SDK
	// tries PAT first, so the env token wins; with auth_type pinned to a
	// non-pat type the SDK uses only that strategy, so the env token is ignored.
	if s.EnvToken != "" && s.Profile != "" {
		if s.ExplicitAuthType != "" && !strings.EqualFold(s.ExplicitAuthType, "pat") {
			out = append(out, Finding{
				Check:  checkNameResolution,
				Level:  LevelWarn,
				Title:  "DATABRICKS_TOKEN is set but ignored",
				Detail: fmt.Sprintf("Profile %q pins auth_type=%s, so the CLI uses that and ignores the personal access token in DATABRICKS_TOKEN.", s.Profile, s.ExplicitAuthType),
				Fix:    "unset DATABRICKS_TOKEN, or remove auth_type to use the token",
			})
		} else {
			out = append(out, Finding{
				Check:  checkNameResolution,
				Level:  LevelWarn,
				Title:  "DATABRICKS_TOKEN overrides the profile credentials",
				Detail: fmt.Sprintf("DATABRICKS_TOKEN is set, so the CLI authenticates with that personal access token instead of profile %q's configured credentials.", s.Profile),
				Fix:    "unset DATABRICKS_TOKEN to use the profile's credentials",
			})
		}
	}

	out = append(out, checkBundleDivergence(s)...)

	return out
}

// checkBundleDivergence flags a databricks.yml bundle whose workspace settings
// would route bundle commands to different credentials than the doctor
// resolved for normal commands.
func checkBundleDivergence(s Snapshot) []Finding {
	if s.BundlePath == "" {
		return nil
	}

	// A value that still contains a "${...}" reference, or a target-specific
	// override, is not resolved here; report the bundle's presence rather than
	// risk a wrong comparison.
	unevaluated := func(v string) bool { return strings.Contains(v, "${") }

	switch {
	case s.BundleProfile != "" && !unevaluated(s.BundleProfile) && s.BundleProfile != s.Profile:
		return []Finding{{
			Check:  checkNameResolution,
			Level:  LevelWarn,
			Title:  "A bundle pins a different profile",
			Detail: fmt.Sprintf("%s sets workspace.profile=%q; bundle commands (databricks bundle ...) use it instead of profile %q.", s.BundlePath, s.BundleProfile, emptyDash(s.Profile)),
		}}
	case s.BundleHost != "" && !unevaluated(s.BundleHost) && !sameHost(s.BundleHost, s.Host):
		return []Finding{{
			Check:  checkNameResolution,
			Level:  LevelWarn,
			Title:  "A bundle pins a different host",
			Detail: fmt.Sprintf("%s sets workspace.host=%s; bundle commands use it instead of %s.", s.BundlePath, s.BundleHost, emptyDash(s.Host)),
		}}
	default:
		return []Finding{{
			Check:  checkNameResolution,
			Level:  LevelOK,
			Title:  "A bundle is present in the working directory",
			Detail: s.BundlePath + ": bundle commands resolve auth from the bundle and selected target, which the doctor does not fully evaluate. Run `databricks bundle validate` to see the bundle's resolved host.",
		}}
	}
}

// checkHost validates that the resolved host matches the kind of command the
// user is likely running and that account scope has an account id.
func checkHost(s Snapshot) []Finding {
	if s.Host == "" {
		return []Finding{{
			Check:  checkNameHost,
			Level:  LevelError,
			Title:  "No host resolved",
			Detail: "A host is required to authenticate.",
			Fix:    "databricks auth login --host <workspace-url>",
		}}
	}

	switch s.HostType {
	case HostTypeAccount:
		if s.AccountID == "" {
			return []Finding{{
				Check:  checkNameHost,
				Level:  LevelError,
				Title:  "Account console host without an account id",
				Detail: s.Host + " is the account console, but no account_id is set. Account commands need account_id; workspace commands need a workspace URL.",
				Fix:    "databricks auth login --host <account-console-url> --account-id <account-id>",
			}}
		}
		return []Finding{{
			Check:  checkNameHost,
			Level:  LevelOK,
			Title:  "Host is the account console",
			Detail: fmt.Sprintf("%s with account_id=%s. Workspace-scoped commands will fail against this host.", s.Host, s.AccountID),
		}}
	case HostTypeUnified:
		return []Finding{{
			Check:  checkNameHost,
			Level:  LevelOK,
			Title:  "Host is a unified (account + workspace) host",
			Detail: s.Host + " serves both account and workspace APIs.",
		}}
	default:
		// account_id on a workspace host is harmless for workspace commands (it
		// is only used for account-scoped operations), so it is noted, not warned.
		detail := s.Host
		if s.AccountID != "" {
			detail = fmt.Sprintf("%s (account_id=%s is set; used only for account-scoped commands).", s.Host, s.AccountID)
		}
		return []Finding{{
			Check:  checkNameHost,
			Level:  LevelOK,
			Title:  "Host is a workspace URL",
			Detail: detail,
		}}
	}
}

// checkAuthType validates the credentials specific to the active auth type.
func checkAuthType(s Snapshot) []Finding {
	switch strings.ToLower(s.AuthType) {
	case "pat":
		if !s.PATPresent {
			return []Finding{{
				Check: checkNameAuthType,
				Level: LevelError,
				Title: "PAT auth selected but no token is set",
				Fix:   "set DATABRICKS_TOKEN or add token to the profile",
			}}
		}
		if !s.PATLooksValid {
			return []Finding{{
				Check:  checkNameAuthType,
				Level:  LevelWarn,
				Title:  "Token does not look like a Databricks PAT",
				Detail: "Databricks personal access tokens start with \"dapi\".",
			}}
		}
		return []Finding{{
			Check: checkNameAuthType,
			Level: LevelOK,
			Title: "Personal access token is present",
		}}
	case "oauth-m2m":
		if !s.ClientIDPresent || !s.ClientSecretPresent {
			return []Finding{{
				Check:  checkNameAuthType,
				Level:  LevelError,
				Title:  "OAuth M2M selected but client credentials are incomplete",
				Detail: fmt.Sprintf("client_id present=%t, client_secret present=%t.", s.ClientIDPresent, s.ClientSecretPresent),
				Fix:    "set client_id and client_secret (DATABRICKS_CLIENT_ID / DATABRICKS_CLIENT_SECRET)",
			}}
		}
		return []Finding{{
			Check: checkNameAuthType,
			Level: LevelOK,
			Title: "OAuth M2M client credentials are present",
		}}
	default:
		return nil
	}
}

// checkToken inspects the cached U2M token: presence, real expiry, and whether
// a refresh token is available to renew it.
func checkToken(s Snapshot) []Finding {
	var out []Finding

	if !s.Token.Found {
		f := Finding{
			Check: checkNameToken,
			Level: LevelError,
			Title: "No cached OAuth token",
			Fix:   loginFix(s.Profile),
		}
		switch {
		case s.Token.NotFoundHint != "":
			f.Detail = s.Token.NotFoundHint
		case s.Token.LookupErr != "":
			f.Detail = "Token cache lookup failed: " + s.Token.LookupErr
		}
		out = append(out, f)
	} else {
		expired := !s.Token.Expiry.IsZero() && !s.Token.Expiry.After(s.Now)
		switch {
		case expired && s.Token.HasRefresh:
			// The cached access token expires hourly; with a refresh token this
			// is the normal state and renews transparently on the next call, so
			// it is not a problem worth flagging.
			out = append(out, Finding{
				Check:  checkNameToken,
				Level:  LevelOK,
				Title:  "Cached access token expired; will auto-refresh",
				Detail: fmt.Sprintf("Expired at %s, but a refresh token is present, so the CLI renews it on the next call.", formatExpiry(s.Token.Expiry, s.Now)),
			})
		case expired && !s.Token.HasRefresh:
			out = append(out, Finding{
				Check:  checkNameToken,
				Level:  LevelError,
				Title:  "Access token expired and no refresh token is present",
				Detail: fmt.Sprintf("Expired at %s and cannot be refreshed silently.", formatExpiry(s.Token.Expiry, s.Now)),
				Fix:    loginFix(s.Profile),
			})
		default:
			detail := fmt.Sprintf("Valid until %s.", formatExpiry(s.Token.Expiry, s.Now))
			if !s.Token.HasRefresh {
				detail += " No refresh token present (offline_access not granted), so it will not refresh after expiry."
			}
			out = append(out, Finding{
				Check:  checkNameToken,
				Level:  LevelOK,
				Title:  "Cached OAuth token is valid",
				Detail: detail,
			})
		}
	}

	if s.KeyringReachable != nil && !*s.KeyringReachable {
		out = append(out, Finding{
			Check:  checkNameToken,
			Level:  LevelWarn,
			Title:  "OS keyring is not reachable",
			Detail: fmt.Sprintf("Token storage is %q but the keyring probe failed: %s. Token reads may fail.", s.StorageMode, emptyDash(s.KeyringProbeErr)),
		})
	}

	return out
}

// checkScopes reports whether the profile is downscoped and explains the fix
// for the OAuth scope errors downscoping produces.
func checkScopes(s Snapshot) []Finding {
	if len(s.ConfiguredScopes) == 0 || containsScope(s.ConfiguredScopes, scopeAllApis) {
		return []Finding{{
			Check:  checkNameScopes,
			Level:  LevelOK,
			Title:  "No OAuth scope restriction",
			Detail: "The profile requests all-apis; commands are not limited by token scope.",
		}}
	}

	// Remediation must remove the restriction, not re-request the same scopes
	// (that would leave the profile downscoped). all-apis clears it; the user
	// can instead pass the specific scope they need.
	return []Finding{{
		Check:  checkNameScopes,
		Level:  LevelWarn,
		Title:  "Profile is downscoped",
		Detail: fmt.Sprintf("OAuth token is limited to scopes: %s. Commands needing another scope fail with an OAuth access_denied error.", joinScopes(s.ConfiguredScopes)),
		Fix:    scopeFixCommand(s.Profile, []string{scopeAllApis}) + " (or add the specific scope you need)",
	}}
}

// checkClock warns when the local clock has drifted far enough from the server
// to break token validity windows (a cause of cryptic auth failures).
func checkClock(s Snapshot) []Finding {
	skew := s.ClockSkew
	if skew < 0 {
		skew = -skew
	}
	if skew < clockSkewThreshold {
		return []Finding{{
			Check: checkNameClock,
			Level: LevelOK,
			Title: "Local clock is in sync with the server",
		}}
	}
	return []Finding{{
		Check:  checkNameClock,
		Level:  LevelWarn,
		Title:  "Local clock differs from the server",
		Detail: fmt.Sprintf("Local time is off by %s from the server; large skew can break token validation. Sync your system clock (NTP).", roundDuration(skew)),
	}}
}

// checkConnectivity interprets the best-effort connectivity probe, classifying
// OAuth scope errors so the doctor explains downscoping failures the SDK hits.
func checkConnectivity(s Snapshot) []Finding {
	if s.ProbeErr == nil {
		title := "Authenticated successfully"
		if s.ProbeUser != "" {
			title = "Authenticated as " + s.ProbeUser
		}
		return []Finding{{
			Check: checkNameConnectivity,
			Level: LevelOK,
			Title: title,
		}}
	}

	if kind, scopes, ok := ClassifyScopeError(s.ProbeErr); ok {
		return []Finding{{
			Check:  checkNameConnectivity,
			Level:  LevelError,
			Title:  scopeErrorTitle(kind),
			Detail: scopeErrorDetail(kind, scopes, s.ProbeErr),
			Fix:    scopeFixCommand(s.Profile, scopeFixScopes(kind, scopes, s.ConfiguredScopes)),
		}}
	}

	return []Finding{{
		Check:  checkNameConnectivity,
		Level:  LevelError,
		Title:  "Authentication check failed",
		Detail: s.ProbeErr.Error(),
		Fix:    loginFix(s.Profile),
	}}
}

// loginFix builds the re-login command for a profile. The profile name is
// shell-quoted so a name with spaces or metacharacters stays copy-paste-safe.
func loginFix(profile string) string {
	if profile == "" {
		return "databricks auth login"
	}
	return "databricks auth login --profile " + shellquote.BashArg(profile)
}

// emptyDash renders empty strings as a dash for readable output.
func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// sameHost reports whether two host strings point at the same host, ignoring
// scheme, case, trailing slash, and query/fragment.
func sameHost(a, b string) bool {
	return canonicalHost(a) == canonicalHost(b)
}

func canonicalHost(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	if h == "" {
		return ""
	}
	if !strings.Contains(h, "://") {
		h = "https://" + h
	}
	u, err := url.Parse(h)
	if err != nil {
		return strings.TrimSuffix(h, "/")
	}
	// Drop an explicit default port so ":443"/":80" compares equal to no port.
	if port := u.Port(); (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		return u.Hostname()
	}
	return u.Host
}
