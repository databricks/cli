// Package authdoctor diagnoses why CLI authentication is broken (or about to
// break) and prints the exact command to fix it.
//
// The package is split in two: Analyze is a pure function over a Snapshot of
// locally observable state (resolved config, cached token, keyring, env vars),
// so the verdict is deterministic and unit-testable. The cmd subpackage owns
// all the I/O that fills the Snapshot. This mirrors the "trace then narrate"
// design: the engine produces the trace, the verdict never depends on a model.
package authdoctor

import (
	"fmt"
	"strings"
	"time"
)

// AuthTypeDatabricksCLI is the SDK auth type for U2M (OAuth) login. Only this
// auth type uses the CLI's token cache, keyring, and downscoping scopes.
const AuthTypeDatabricksCLI = "databricks-cli"

// Profile-source labels. These describe where the effective profile was
// resolved from and double as the human-readable ProfileSource shown in output.
// They are exported so the cmd layer sets the same labels Analyze compares on.
const (
	SourceFlag           = "--profile flag"
	SourceEnvProfile     = "DATABRICKS_CONFIG_PROFILE"
	SourceDefaultProfile = "[__settings__].default_profile"
	SourceDefaultSection = "[DEFAULT] section"
	SourceNone           = "no profile (host/env only)"
)

// Level ranks a finding from best to worst. Analyze sorts and summarizes on it.
type Level string

const (
	LevelOK    Level = "ok"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// rank orders levels so the worst finding wins the summary.
func (l Level) rank() int {
	switch l {
	case LevelError:
		return 2
	case LevelWarn:
		return 1
	default:
		return 0
	}
}

// Finding is one diagnosis with an optional one-line remediation command.
type Finding struct {
	Check  string `json:"check"`
	Level  Level  `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	// Fix is a copy-pasteable command that resolves the finding. Empty for OK
	// findings and for findings the user cannot fix with a single command.
	Fix string `json:"fix,omitempty"`
}

// TokenState is the result of looking up the cached U2M token for a profile.
// The cmd layer fills it; Analyze only interprets it. It is meaningful only
// when the active auth type is databricks-cli.
type TokenState struct {
	// Found is true when a token was present in the cache.
	Found bool
	// Expiry is the access-token expiry; zero when unknown.
	Expiry time.Time
	// HasRefresh is true when the cached token carries a refresh token. Without
	// one, an expired access token cannot be renewed silently.
	HasRefresh bool
	// LookupErr is the non-not-found error from the cache lookup, if any.
	LookupErr string
	// NotFoundHint carries the storage layer's upgrade-aware hint when the token
	// was not found (e.g. credentials from an older CLI are no longer used).
	NotFoundHint string
}

// Snapshot is the locally observable auth state for a single profile. Every
// field is plain data so Analyze stays pure and testable.
type Snapshot struct {
	// Profile is the effective profile name. Empty when none was resolved.
	Profile string
	// ProfileSource describes where Profile came from (flag, env, default_profile,
	// the [DEFAULT] section, or none).
	ProfileSource string
	// Host, AuthType, AccountID are the resolved effective values. AuthType is
	// the doctor's best-effort effective auth type (inferred when not pinned).
	Host      string
	AuthType  string
	AccountID string
	// ExplicitAuthType is the auth type the config explicitly pins (profile
	// auth_type or DATABRICKS_AUTH_TYPE), empty when none is set. With an
	// explicit auth type the SDK uses only that strategy, so DATABRICKS_TOKEN is
	// ignored unless the pinned type is pat.
	ExplicitAuthType string
	// HostType is the host classification of Host: "workspace" or "account".
	HostType string
	// ConfigError is the EnsureResolved error string, if config did not validate.
	ConfigError string

	// ProfileHost is the host declared for this profile in the config file,
	// before env overrides. Used to detect DATABRICKS_HOST shadowing the profile.
	ProfileHost string
	// ConfiguredScopes are the OAuth scopes persisted on the profile (split).
	ConfiguredScopes []string

	// Bundle* describe a databricks.yml bundle found in the working-directory
	// tree, if any. Bundle commands resolve auth from the bundle, diverging from
	// normal commands. BundleProfile/BundleHost are the top-level workspace
	// settings read best-effort; targets, includes, and variables are NOT
	// expanded, so a value containing "${" is reported as unevaluated.
	BundlePath    string
	BundleProfile string
	BundleHost    string

	// EnvHost and EnvToken hold the auth-related environment variables that the
	// resolution checks compare against the profile.
	EnvHost  string
	EnvToken string

	// PAT / M2M presence, computed without retaining the secret value.
	PATPresent          bool
	PATLooksValid       bool
	ClientIDPresent     bool
	ClientSecretPresent bool

	// Token is the cached U2M token state (databricks-cli auth only).
	Token TokenState
	// StorageMode and StorageSource describe the token cache backend + provenance.
	StorageMode   string
	StorageSource string
	// KeyringReachable is nil when not probed (non-U2M auth), else the probe result.
	KeyringReachable *bool
	KeyringProbeErr  string

	// Probe* hold the best-effort connectivity probe result. ProbeAttempted is
	// false when the probe was skipped (e.g. under --all or no host).
	ProbeAttempted bool
	ProbeUser      string
	ProbeErr       error

	// ClockSkew is local-minus-server time; ClockSkewKnown is false when it
	// could not be measured (e.g. under --all or no network).
	ClockSkew      time.Duration
	ClockSkewKnown bool

	// Now is the reference time for expiry comparisons. Injected for determinism.
	Now time.Time
}

// Report is the full diagnosis for one profile.
type Report struct {
	Profile  string    `json:"profile,omitempty"`
	Host     string    `json:"host,omitempty"`
	AuthType string    `json:"auth_type,omitempty"`
	Overall  Level     `json:"overall"`
	Findings []Finding `json:"findings"`
}

// Analyze runs every check against the snapshot and returns a ranked report.
// It performs no I/O: the verdict is a deterministic function of the snapshot.
func Analyze(s Snapshot) Report {
	r := Report{
		Profile:  s.Profile,
		Host:     s.Host,
		AuthType: s.AuthType,
	}

	r.Findings = append(r.Findings, checkResolution(s)...)
	r.Findings = append(r.Findings, checkHost(s)...)
	r.Findings = append(r.Findings, checkAuthType(s)...)
	if IsU2M(s.AuthType) {
		r.Findings = append(r.Findings, checkToken(s)...)
		r.Findings = append(r.Findings, checkScopes(s)...)
	}
	if s.ClockSkewKnown {
		r.Findings = append(r.Findings, checkClock(s)...)
	}
	if s.ProbeAttempted {
		r.Findings = append(r.Findings, checkConnectivity(s)...)
	}

	r.Overall = LevelOK
	for _, f := range r.Findings {
		if f.Level.rank() > r.Overall.rank() {
			r.Overall = f.Level
		}
	}
	return r
}

// IsU2M reports whether authType is the U2M (OAuth) login that uses the CLI
// token cache, keyring, and downscoping scopes.
func IsU2M(authType string) bool {
	return authType == AuthTypeDatabricksCLI
}

// formatExpiry renders an absolute timestamp plus a relative hint, e.g.
// "2026-06-25 14:00:00 UTC (in 42m)" or "... (expired 2h ago)".
func formatExpiry(expiry, now time.Time) string {
	abs := expiry.UTC().Format("2006-01-02 15:04:05 MST")
	d := expiry.Sub(now)
	// d > 0 (not >=) so the boundary matches checkToken's expiry test
	// (!Expiry.After(Now)): at exactly now, the token is expired.
	if d > 0 {
		return fmt.Sprintf("%s (in %s)", abs, roundDuration(d))
	}
	return fmt.Sprintf("%s (expired %s ago)", abs, roundDuration(-d))
}

// roundDuration trims a duration to a human-friendly unit.
func roundDuration(d time.Duration) string {
	if d >= time.Hour {
		return d.Round(time.Minute).String()
	}
	return d.Round(time.Second).String()
}

// joinScopes renders a scope list for display.
func joinScopes(scopes []string) string {
	return strings.Join(scopes, ", ")
}
