package authdoctor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var refTime = time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)

// healthySnapshot returns a fully working U2M snapshot that Analyze rates OK.
// Tests mutate one aspect to exercise a single check.
func healthySnapshot() Snapshot {
	reachable := true
	return Snapshot{
		Profile:          "DEFAULT",
		ProfileSource:    SourceDefaultProfile,
		Host:             "https://myworkspace.cloud.databricks.test",
		ProfileHost:      "https://myworkspace.cloud.databricks.test",
		HostType:         HostTypeWorkspace,
		AuthType:         AuthTypeDatabricksCLI,
		ConfiguredScopes: nil,
		Token: TokenState{
			Found:      true,
			Expiry:     refTime.Add(time.Hour),
			HasRefresh: true,
		},
		StorageMode:      "secure",
		KeyringReachable: &reachable,
		Now:              refTime,
	}
}

// findOne returns the first finding for a check, requiring it exists.
func findOne(t *testing.T, r Report, check string) Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Check == check {
			return f
		}
	}
	require.Failf(t, "missing finding", "no finding for check %q", check)
	return Finding{}
}

func TestAnalyzeHealthyIsOK(t *testing.T) {
	r := Analyze(healthySnapshot())
	assert.Equal(t, LevelOK, r.Overall)
	for _, f := range r.Findings {
		assert.Equal(t, LevelOK, f.Level, "finding %q should be OK", f.Check)
	}
}

func TestAnalyzeNoProfileNoHost(t *testing.T) {
	s := Snapshot{ProfileSource: SourceNone, Now: refTime}
	r := Analyze(s)
	assert.Equal(t, LevelError, r.Overall)
	f := findOne(t, r, checkNameResolution)
	assert.Equal(t, LevelError, f.Level)
	assert.Contains(t, f.Fix, "databricks auth login")
}

func TestAnalyzeDefaultSectionDoesNotWarn(t *testing.T) {
	// All four resolution paths (normal, api, auth token, bundle) resolve the
	// [DEFAULT] section identically via databrickscfg.ResolveDefaultProfile, so
	// resolving from [DEFAULT] is not a divergence and must not warn.
	s := healthySnapshot()
	s.ProfileSource = SourceDefaultSection
	r := Analyze(s)
	assert.Equal(t, LevelOK, r.Overall)
	for _, f := range r.Findings {
		assert.NotContains(t, f.Detail, "databricks api")
	}
}

func TestAnalyzeEnvHostOverridesProfile(t *testing.T) {
	s := healthySnapshot()
	s.EnvHost = "https://other.cloud.databricks.test"
	r := Analyze(s)
	assert.Equal(t, LevelWarn, r.Overall)

	var titles []string
	for _, f := range r.Findings {
		if f.Check == checkNameResolution && f.Level == LevelWarn {
			titles = append(titles, f.Title)
		}
	}
	assert.Contains(t, titles, "DATABRICKS_HOST overrides the profile host")
}

func TestAnalyzeBundleDivergence(t *testing.T) {
	t.Run("different profile warns", func(t *testing.T) {
		s := healthySnapshot()
		s.BundlePath = "/work/databricks.yml"
		s.BundleProfile = "prod"
		r := Analyze(s)
		assert.Equal(t, LevelWarn, r.Overall)
		var found bool
		for _, f := range r.Findings {
			if f.Title == "A bundle pins a different profile" {
				assert.Contains(t, f.Detail, "prod")
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("unevaluated variable does not warn", func(t *testing.T) {
		s := healthySnapshot()
		s.BundlePath = "/work/databricks.yml"
		s.BundleProfile = "${var.profile}"
		r := Analyze(s)
		// Falls through to the informational "bundle is present" finding.
		assert.Equal(t, LevelOK, r.Overall)
		var found bool
		for _, f := range r.Findings {
			if f.Title == "A bundle is present in the working directory" {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("matching profile is informational", func(t *testing.T) {
		s := healthySnapshot()
		s.BundlePath = "/work/databricks.yml"
		s.BundleProfile = s.Profile
		r := Analyze(s)
		assert.Equal(t, LevelOK, r.Overall)
	})
}

func TestAnalyzeEnvTokenOverridesProfile(t *testing.T) {
	s := healthySnapshot()
	s.EnvToken = "dapi123"
	r := Analyze(s)

	var found bool
	for _, f := range r.Findings {
		if f.Title == "DATABRICKS_TOKEN overrides the profile credentials" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestAnalyzeEnvTokenIgnoredWhenAuthTypePinned(t *testing.T) {
	s := healthySnapshot()
	s.EnvToken = "dapi123"
	s.ExplicitAuthType = "databricks-cli"
	r := Analyze(s)

	var ignored, overrides bool
	for _, f := range r.Findings {
		switch f.Title {
		case "DATABRICKS_TOKEN is set but ignored":
			ignored = true
		case "DATABRICKS_TOKEN overrides the profile credentials":
			overrides = true
		}
	}
	assert.True(t, ignored, "pinned non-pat auth_type should report the token as ignored")
	assert.False(t, overrides)
}

func TestAnalyzeTokenStates(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Snapshot)
		wantLevel Level
		wantFix   bool
	}{
		{
			name:      "expired with refresh",
			mutate:    func(s *Snapshot) { s.Token.Expiry = refTime.Add(-time.Hour); s.Token.HasRefresh = true },
			wantLevel: LevelWarn,
		},
		{
			name:      "expired no refresh",
			mutate:    func(s *Snapshot) { s.Token.Expiry = refTime.Add(-time.Hour); s.Token.HasRefresh = false },
			wantLevel: LevelError,
			wantFix:   true,
		},
		{
			name:      "missing",
			mutate:    func(s *Snapshot) { s.Token = TokenState{} },
			wantLevel: LevelError,
			wantFix:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := healthySnapshot()
			tt.mutate(&s)
			r := Analyze(s)
			f := findOne(t, r, checkNameToken)
			assert.Equal(t, tt.wantLevel, f.Level)
			if tt.wantFix {
				assert.Contains(t, f.Fix, "databricks auth login --profile DEFAULT")
			}
		})
	}
}

func TestAnalyzeValidTokenNoRefreshNote(t *testing.T) {
	s := healthySnapshot()
	s.Token.HasRefresh = false
	r := Analyze(s)
	f := findOne(t, r, checkNameToken)
	assert.Equal(t, LevelOK, f.Level)
	assert.Contains(t, f.Detail, "will not refresh")
}

func TestAnalyzeKeyringUnreachable(t *testing.T) {
	s := healthySnapshot()
	reachable := false
	s.KeyringReachable = &reachable
	s.KeyringProbeErr = "keyring locked"
	r := Analyze(s)
	assert.Equal(t, LevelWarn, r.Overall)

	var found bool
	for _, f := range r.Findings {
		if f.Title == "OS keyring is not reachable" {
			assert.Contains(t, f.Detail, "keyring locked")
			found = true
		}
	}
	assert.True(t, found)
}

func TestAnalyzeDownscoped(t *testing.T) {
	s := healthySnapshot()
	s.ConfiguredScopes = []string{"jobs", "sql"}
	r := Analyze(s)
	f := findOne(t, r, checkNameScopes)
	assert.Equal(t, LevelWarn, f.Level)
	assert.Contains(t, f.Detail, "jobs, sql")
	// The fix must remove the restriction, not re-request the same scopes.
	assert.Contains(t, f.Fix, "--scopes 'all-apis'")
	assert.NotContains(t, f.Fix, "'jobs,sql'")
}

func TestAnalyzeAllApisIsOK(t *testing.T) {
	s := healthySnapshot()
	s.ConfiguredScopes = []string{"all-apis"}
	r := Analyze(s)
	assert.Equal(t, LevelOK, findOne(t, r, checkNameScopes).Level)
}

func TestAnalyzeAccountHost(t *testing.T) {
	t.Run("no account id", func(t *testing.T) {
		s := healthySnapshot()
		s.HostType = HostTypeAccount
		s.Host = "https://accounts.cloud.databricks.test"
		r := Analyze(s)
		f := findOne(t, r, checkNameHost)
		assert.Equal(t, LevelError, f.Level)
		assert.Contains(t, f.Fix, "--account-id")
	})
	t.Run("with account id", func(t *testing.T) {
		s := healthySnapshot()
		s.HostType = HostTypeAccount
		s.Host = "https://accounts.cloud.databricks.test"
		s.AccountID = "abc-123"
		r := Analyze(s)
		assert.Equal(t, LevelOK, findOne(t, r, checkNameHost).Level)
	})
}

func TestAnalyzeAccountIDOnWorkspaceHost(t *testing.T) {
	s := healthySnapshot()
	s.AccountID = "abc-123"
	r := Analyze(s)
	f := findOne(t, r, checkNameHost)
	assert.Equal(t, LevelWarn, f.Level)
	assert.Contains(t, f.Title, "account_id set on a workspace host")
}

func TestAnalyzeAuthTypePAT(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Snapshot)
		wantLevel Level
	}{
		{"valid", func(s *Snapshot) { s.PATPresent = true; s.PATLooksValid = true }, LevelOK},
		{"missing", func(s *Snapshot) { s.PATPresent = false }, LevelError},
		{"malformed", func(s *Snapshot) { s.PATPresent = true; s.PATLooksValid = false }, LevelWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := healthySnapshot()
			s.AuthType = "pat"
			tt.mutate(&s)
			r := Analyze(s)
			assert.Equal(t, tt.wantLevel, findOne(t, r, checkNameAuthType).Level)
		})
	}
}

func TestAnalyzeAuthTypeM2M(t *testing.T) {
	s := healthySnapshot()
	s.AuthType = "oauth-m2m"
	s.ClientIDPresent = true
	s.ClientSecretPresent = false
	r := Analyze(s)
	f := findOne(t, r, checkNameAuthType)
	assert.Equal(t, LevelError, f.Level)
	assert.Contains(t, f.Fix, "client_id and client_secret")
}

func TestAnalyzeSkipsTokenChecksForNonU2M(t *testing.T) {
	s := healthySnapshot()
	s.AuthType = "pat"
	s.PATPresent = true
	s.PATLooksValid = true
	s.Token = TokenState{} // would be an error finding under U2M
	r := Analyze(s)
	for _, f := range r.Findings {
		assert.NotEqual(t, checkNameToken, f.Check, "PAT auth should not run token checks")
		assert.NotEqual(t, checkNameScopes, f.Check, "PAT auth should not run scope checks")
	}
}

func TestAnalyzeUnifiedHost(t *testing.T) {
	s := healthySnapshot()
	s.HostType = HostTypeUnified
	r := Analyze(s)
	f := findOne(t, r, checkNameHost)
	assert.Equal(t, LevelOK, f.Level)
	assert.Contains(t, f.Title, "unified")
}

func TestAnalyzeClockSkew(t *testing.T) {
	t.Run("in sync", func(t *testing.T) {
		s := healthySnapshot()
		s.ClockSkewKnown = true
		s.ClockSkew = 30 * time.Second
		r := Analyze(s)
		assert.Equal(t, LevelOK, findOne(t, r, checkNameClock).Level)
	})
	t.Run("skewed", func(t *testing.T) {
		s := healthySnapshot()
		s.ClockSkewKnown = true
		s.ClockSkew = -10 * time.Minute
		r := Analyze(s)
		f := findOne(t, r, checkNameClock)
		assert.Equal(t, LevelWarn, f.Level)
		assert.Contains(t, f.Detail, "10m0s")
	})
	t.Run("not measured skips the check", func(t *testing.T) {
		s := healthySnapshot()
		r := Analyze(s)
		for _, f := range r.Findings {
			assert.NotEqual(t, checkNameClock, f.Check)
		}
	})
}

func TestFormatExpiry(t *testing.T) {
	assert.Contains(t, formatExpiry(refTime.Add(90*time.Minute), refTime), "in 1h30m0s")
	assert.Contains(t, formatExpiry(refTime.Add(-2*time.Hour), refTime), "expired 2h0m0s ago")
	assert.Contains(t, formatExpiry(refTime.Add(time.Hour), refTime), "2026-06-25 13:00:00 UTC")
}
