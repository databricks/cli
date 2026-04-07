package auth

import (
	"errors"
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
		wantErr     error
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
			wantErr: errNoProfileFound,
		},
		{
			name:        "http prefix",
			profiles:    profile.Profiles{},
			arg:         "http://localhost:8080",
			wantProfile: "",
			wantHost:    "http://localhost:8080",
		},
		{
			name:        "host:port without dots or scheme",
			profiles:    profile.Profiles{},
			arg:         "localhost:8080",
			wantProfile: "",
			wantHost:    "localhost:8080",
		},
		{
			name:     "empty profiles error",
			profiles: profile.Profiles{},
			arg:      "myprofile",
			wantErr:  errNoProfileFound,
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
			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr), "expected %v, got %v", tc.wantErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantProfile, profileName)
			assert.Equal(t, tc.wantHost, host)
		})
	}
}

func TestResolveHostToProfileMatchesOneProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "staging", Host: "https://staging.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	resolved, err := resolveHostToProfile(ctx, "https://dev.cloud.databricks.com", profiler)
	require.NoError(t, err)
	assert.Equal(t, "dev", resolved)
}

func TestResolveHostToProfileMatchesMultipleProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev1", Host: "https://shared.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "dev2", Host: "https://shared.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	_, err := resolveHostToProfile(ctx, "https://shared.cloud.databricks.com", profiler)
	assert.ErrorContains(t, err, "multiple profiles found matching host")
	assert.ErrorContains(t, err, "dev1")
	assert.ErrorContains(t, err, "dev2")
}

func TestResolveHostToProfileMatchesNothing(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "staging", Host: "https://staging.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	_, err := resolveHostToProfile(ctx, "https://unknown.cloud.databricks.com", profiler)
	assert.ErrorContains(t, err, `no profile found matching host "https://unknown.cloud.databricks.com"`)
	assert.ErrorContains(t, err, "dev")
	assert.ErrorContains(t, err, "staging")
}

func TestResolveHostToProfileCanonicalizesHost(t *testing.T) {
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	cases := []struct {
		name string
		arg  string
	}{
		{name: "canonical URL", arg: "https://dev.cloud.databricks.com"},
		{name: "trailing slash", arg: "https://dev.cloud.databricks.com/"},
		{name: "no scheme", arg: "dev.cloud.databricks.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			resolved, err := resolveHostToProfile(ctx, tc.arg, profiler)
			require.NoError(t, err)
			assert.Equal(t, "dev", resolved)
		})
	}
}
