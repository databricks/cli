package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceConfigPopulatesHostAndProfile(t *testing.T) {
	cases := []struct {
		name        string
		host        string
		profile     string
		wantHost    string
		wantProfile string
	}{
		{
			name:        "host only",
			host:        "https://abc.cloud.databricks.com",
			wantHost:    "https://abc.cloud.databricks.com",
			wantProfile: "",
		},
		{
			name:        "profile only",
			profile:     "PROFILE-A",
			wantHost:    "",
			wantProfile: "PROFILE-A",
		},
		{
			name:        "host and profile",
			host:        "https://abc.cloud.databricks.com",
			profile:     "PROFILE-A",
			wantHost:    "https://abc.cloud.databricks.com",
			wantProfile: "PROFILE-A",
		},
		{
			name:        "empty",
			wantHost:    "",
			wantProfile: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := Workspace{Host: tc.host, Profile: tc.profile}
			cfg := w.Config()
			assert.Equal(t, tc.wantHost, cfg.Host)
			assert.Equal(t, tc.wantProfile, cfg.Profile)
		})
	}
}
