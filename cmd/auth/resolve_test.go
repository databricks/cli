package auth

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePositionalArg(t *testing.T) {
	cases := []struct {
		name        string
		profiles    profile.Profiles
		arg         string
		wantProfile string
		wantHost    string
		wantErr     string
	}{
		{
			name: "matches profile",
			profiles: profile.Profiles{
				{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
			},
			arg:         "logfood",
			wantProfile: "logfood",
			wantHost:    "",
		},
		{
			name: "falls back to https host",
			profiles: profile.Profiles{
				{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
			},
			arg:         "https://other.cloud.databricks.com",
			wantProfile: "",
			wantHost:    "https://other.cloud.databricks.com",
		},
		{
			name:        "falls back to host with dot",
			profiles:    profile.Profiles{},
			arg:         "my-workspace.cloud.databricks.com",
			wantProfile: "",
			wantHost:    "my-workspace.cloud.databricks.com",
		},
		{
			name: "errors for non-host non-profile",
			profiles: profile.Profiles{
				{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
			},
			arg:     "e2-logfood",
			wantErr: `no profile named "e2-logfood" found`,
		},
		{
			name:        "http prefix",
			profiles:    profile.Profiles{},
			arg:         "http://localhost:8080",
			wantProfile: "",
			wantHost:    "http://localhost:8080",
		},
		{
			name:     "empty profiles error",
			profiles: profile.Profiles{},
			arg:      "myprofile",
			wantErr:  `no profile named "myprofile" found`,
		},
		{
			name: "profile with dot in name",
			profiles: profile.Profiles{
				{Name: "default.dev", Host: "https://dev.cloud.databricks.com"},
			},
			arg:         "default.dev",
			wantProfile: "default.dev",
			wantHost:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			profiler := profile.InMemoryProfiler{Profiles: tc.profiles}
			profileName, host, err := resolvePositionalArg(ctx, tc.arg, profiler)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantProfile, profileName)
			assert.Equal(t, tc.wantHost, host)
		})
	}
}
