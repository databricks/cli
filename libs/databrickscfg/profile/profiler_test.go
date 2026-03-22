package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithHost(t *testing.T) {
	cases := []struct {
		name        string
		inputHost   string
		profileHost string
		want        bool
	}{
		{
			name:        "exact match with scheme",
			inputHost:   "https://myworkspace.cloud.databricks.com",
			profileHost: "https://myworkspace.cloud.databricks.com",
			want:        true,
		},
		{
			name:        "match without scheme on input",
			inputHost:   "myworkspace.cloud.databricks.com",
			profileHost: "https://myworkspace.cloud.databricks.com",
			want:        true,
		},
		{
			name:        "match stripping trailing slash",
			inputHost:   "https://myworkspace.cloud.databricks.com/",
			profileHost: "https://myworkspace.cloud.databricks.com",
			want:        true,
		},
		{
			name:        "match stripping path",
			inputHost:   "https://myworkspace.cloud.databricks.com/some/path?query=1",
			profileHost: "https://myworkspace.cloud.databricks.com",
			want:        true,
		},
		{
			name:        "no match different host",
			inputHost:   "https://other.cloud.databricks.com",
			profileHost: "https://myworkspace.cloud.databricks.com",
			want:        false,
		},
		{
			name:        "empty host on profile skipped",
			inputHost:   "https://myworkspace.cloud.databricks.com",
			profileHost: "",
			want:        false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Profile{Host: c.profileHost}
			fn := WithHost(c.inputHost)
			assert.Equal(t, c.want, fn(p))
		})
	}
}

func TestMatchProfileNames(t *testing.T) {
	fn := MatchProfileNames("dev", "staging")

	assert.True(t, fn(Profile{Name: "dev"}))
	assert.True(t, fn(Profile{Name: "staging"}))
	assert.False(t, fn(Profile{Name: "production"}))
	assert.False(t, fn(Profile{Name: ""}))
}

func TestMatchProfileNamesEmpty(t *testing.T) {
	fn := MatchProfileNames()
	assert.False(t, fn(Profile{Name: "anything"}))
}

func TestWithHostAndAccountID(t *testing.T) {
	cases := []struct {
		name             string
		inputHost        string
		inputAccountID   string
		profileHost      string
		profileAccountID string
		want             bool
	}{
		{
			name:             "same host same account ID",
			inputHost:        "https://accounts.cloud.databricks.com",
			inputAccountID:   "abc123",
			profileHost:      "https://accounts.cloud.databricks.com",
			profileAccountID: "abc123",
			want:             true,
		},
		{
			name:             "same host different account ID",
			inputHost:        "https://accounts.cloud.databricks.com",
			inputAccountID:   "abc123",
			profileHost:      "https://accounts.cloud.databricks.com",
			profileAccountID: "xyz789",
			want:             false,
		},
		{
			name:             "different host same account ID",
			inputHost:        "https://other.cloud.databricks.com",
			inputAccountID:   "abc123",
			profileHost:      "https://accounts.cloud.databricks.com",
			profileAccountID: "abc123",
			want:             false,
		},
		{
			name:             "empty host on profile skipped",
			inputHost:        "https://accounts.cloud.databricks.com",
			inputAccountID:   "abc123",
			profileHost:      "",
			profileAccountID: "abc123",
			want:             false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Profile{Host: c.profileHost, AccountID: c.profileAccountID}
			fn := WithHostAndAccountID(c.inputHost, c.inputAccountID)
			assert.Equal(t, c.want, fn(p))
		})
	}
}

func TestWithHostAccountIDAndWorkspaceID(t *testing.T) {
	cases := []struct {
		name               string
		inputHost          string
		inputAccountID     string
		inputWorkspaceID   string
		profileHost        string
		profileAccountID   string
		profileWorkspaceID string
		want               bool
	}{
		{
			name:               "all three match",
			inputHost:          "https://spog.example.com",
			inputAccountID:     "acc-1",
			inputWorkspaceID:   "ws-1",
			profileHost:        "https://spog.example.com",
			profileAccountID:   "acc-1",
			profileWorkspaceID: "ws-1",
			want:               true,
		},
		{
			name:               "different workspace_id",
			inputHost:          "https://spog.example.com",
			inputAccountID:     "acc-1",
			inputWorkspaceID:   "ws-1",
			profileHost:        "https://spog.example.com",
			profileAccountID:   "acc-1",
			profileWorkspaceID: "ws-2",
			want:               false,
		},
		{
			name:               "different account_id",
			inputHost:          "https://spog.example.com",
			inputAccountID:     "acc-1",
			inputWorkspaceID:   "ws-1",
			profileHost:        "https://spog.example.com",
			profileAccountID:   "acc-2",
			profileWorkspaceID: "ws-1",
			want:               false,
		},
		{
			name:               "different host",
			inputHost:          "https://other.example.com",
			inputAccountID:     "acc-1",
			inputWorkspaceID:   "ws-1",
			profileHost:        "https://spog.example.com",
			profileAccountID:   "acc-1",
			profileWorkspaceID: "ws-1",
			want:               false,
		},
		{
			name:               "empty host on profile",
			inputHost:          "https://spog.example.com",
			inputAccountID:     "acc-1",
			inputWorkspaceID:   "ws-1",
			profileHost:        "",
			profileAccountID:   "acc-1",
			profileWorkspaceID: "ws-1",
			want:               false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Profile{Host: c.profileHost, AccountID: c.profileAccountID, WorkspaceID: c.profileWorkspaceID}
			fn := WithHostAccountIDAndWorkspaceID(c.inputHost, c.inputAccountID, c.inputWorkspaceID)
			assert.Equal(t, c.want, fn(p))
		})
	}
}

func TestMatchWorkspaceProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		want    bool
	}{
		{
			name:    "regular workspace (no account_id)",
			profile: Profile{Host: "https://ws.cloud.databricks.com"},
			want:    true,
		},
		{
			name:    "SPOG workspace (has workspace_id)",
			profile: Profile{Host: "https://spog.example.com", AccountID: "acc-1", WorkspaceID: "ws-1"},
			want:    true,
		},
		{
			name:    "legacy unified workspace (has workspace_id and IsUnifiedHost)",
			profile: Profile{Host: "https://unified.example.com", AccountID: "acc-1", WorkspaceID: "ws-1", IsUnifiedHost: true},
			want:    true,
		},
		{
			name:    "regular account profile (has account_id, no workspace_id)",
			profile: Profile{Host: "https://accounts.cloud.databricks.com", AccountID: "acc-1"},
			want:    false,
		},
		{
			name:    "legacy unified account (IsUnifiedHost, no workspace_id)",
			profile: Profile{Host: "https://unified.example.com", AccountID: "acc-1", IsUnifiedHost: true},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchWorkspaceProfiles(tt.profile))
		})
	}
}

func TestMatchAccountProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		want    bool
	}{
		{
			name:    "regular account profile",
			profile: Profile{Host: "https://accounts.cloud.databricks.com", AccountID: "acc-1"},
			want:    true,
		},
		{
			name:    "SPOG account profile (account_id, no workspace_id)",
			profile: Profile{Host: "https://spog.example.com", AccountID: "acc-1"},
			want:    true,
		},
		{
			name:    "legacy unified account profile",
			profile: Profile{Host: "https://unified.example.com", AccountID: "acc-1", IsUnifiedHost: true},
			want:    true,
		},
		{
			name:    "SPOG workspace profile (has workspace_id)",
			profile: Profile{Host: "https://spog.example.com", AccountID: "acc-1", WorkspaceID: "ws-1"},
			want:    false,
		},
		{
			name:    "regular workspace (no account_id)",
			profile: Profile{Host: "https://ws.cloud.databricks.com"},
			want:    false,
		},
		{
			name:    "no host",
			profile: Profile{AccountID: "acc-1"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchAccountProfiles(tt.profile))
		})
	}
}
