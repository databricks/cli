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
