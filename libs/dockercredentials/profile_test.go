package dockercredentials_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	workspaceID   = "123456789"
	workspaceHost = "https://workspace.cloud.databricks.test"
)

type invalidRefreshTokenError struct {
	error
}

func (e invalidRefreshTokenError) As(target any) bool {
	invalidRefreshToken, ok := target.(**u2m.InvalidRefreshTokenError)
	if !ok {
		return false
	}
	*invalidRefreshToken = new(u2m.InvalidRefreshTokenError)
	return true
}

func workspaceProfile(name string) profile.Profile {
	return profile.Profile{
		Name:        name,
		Host:        workspaceHost,
		WorkspaceID: workspaceID,
		AuthType:    auth.AuthTypeDatabricksCli,
	}
}

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile profile.Profile
		wantErr string
	}{
		{name: "workspace", profile: workspaceProfile("workspace")},
		{
			name: "client credentials",
			profile: profile.Profile{
				Name:                 "m2m",
				Host:                 workspaceHost,
				WorkspaceID:          workspaceID,
				HasClientCredentials: true,
			},
			wantErr: `profile "m2m" uses client credentials. Docker credential helper requires a profile created by databricks auth login`,
		},
		{
			name: "unsupported auth type",
			profile: profile.Profile{
				Name:        "pat",
				Host:        workspaceHost,
				WorkspaceID: workspaceID,
				AuthType:    auth.AuthTypePat,
			},
			wantErr: `profile "pat" uses auth_type "pat". Docker credential helper requires a profile created by databricks auth login`,
		},
		{
			name: "classic account",
			profile: profile.Profile{
				Name:        "account",
				Host:        "https://accounts.cloud.databricks.test",
				AccountID:   "account-id",
				WorkspaceID: workspaceID,
				AuthType:    auth.AuthTypeDatabricksCli,
			},
			wantErr: `profile "account" does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile`,
		},
		{
			name: "unified account",
			profile: profile.Profile{
				Name:      "account",
				Host:      workspaceHost,
				AccountID: "account-id",
				AuthType:  auth.AuthTypeDatabricksCli,
			},
			wantErr: `profile "account" does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dockercredentials.ValidateProfile(tt.profile)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestLoadProfile(t *testing.T) {
	valid := workspaceProfile("workspace")
	tests := []struct {
		name     string
		profiles profile.Profiles
		profile  string
		want     profile.Profile
		wantErr  string
	}{
		{name: "loads valid profile", profiles: profile.Profiles{valid}, profile: valid.Name, want: valid},
		{name: "profile not found", profile: "missing", wantErr: `profile "missing" not found`},
		{
			name: "invalid profile",
			profiles: profile.Profiles{{
				Name:                 "m2m",
				Host:                 workspaceHost,
				WorkspaceID:          workspaceID,
				HasClientCredentials: true,
			}},
			profile: "m2m",
			wantErr: `profile "m2m" uses client credentials. Docker credential helper requires a profile created by databricks auth login`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dockercredentials.LoadProfile(t.Context(), profile.InMemoryProfiler{Profiles: tt.profiles}, tt.profile)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProfileForRegistry(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: workspaceID,
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	workspace := workspaceProfile("workspace")
	m2m := profile.Profile{
		Name:                 "m2m",
		Host:                 workspaceHost,
		WorkspaceID:          workspaceID,
		HasClientCredentials: true,
	}
	tests := []struct {
		name     string
		profiles profile.Profiles
		recorded string
		want     string
		wantErr  string
	}{
		{name: "selects workspace profile", profiles: profile.Profiles{workspace}, want: workspace.Name},
		{
			name:     "duplicate workspace ID",
			profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			wantErr:  "multiple Databricks profiles match workspace ID 123456789: prod and dev. Select the profile for Docker with: databricks auth docker configure <profile-name>",
		},
		{
			name:     "ignores unsupported duplicate",
			profiles: profile.Profiles{workspace, m2m},
			want:     workspace.Name,
		},
		{
			name:    "no matching profile",
			wantErr: "no Databricks profile found for workspace ID 123456789 from registry host 123456789.container.us-west-2.cloud.databricks.test. Run databricks auth login --host <workspace-url> and set workspace_id for that profile",
		},
		{
			name:     "selects recorded profile",
			profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			recorded: "dev",
			want:     "dev",
		},
		{
			name:     "recorded profile is missing",
			profiles: profile.Profiles{workspace},
			recorded: "gone",
			wantErr:  `profile "gone" selected for Docker registry 123456789.container.us-west-2.cloud.databricks.test is missing or no longer targets workspace ID 123456789. Select the profile for Docker with: databricks auth docker configure <profile-name>`,
		},
		{
			name:     "recorded profile is unsupported",
			profiles: profile.Profiles{workspace, m2m},
			recorded: "m2m",
			wantErr:  `profile "m2m" uses client credentials. Docker credential helper requires a profile created by databricks auth login`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dockercredentials.ProfileForRegistry(t.Context(), profile.InMemoryProfiler{Profiles: tt.profiles}, registry, tt.recorded)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}

func TestRewriteProfileErrorPreservesInvalidRefreshTokenCause(t *testing.T) {
	cause := invalidRefreshTokenError{error: errors.New("raw token refresh error")}
	err := dockercredentials.RewriteProfileError(t.Context(), workspaceProfile("workspace"), fmt.Errorf("refresh token: %w", cause))

	assert.EqualError(t, err, `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile workspace`)
	assert.ErrorIs(t, err, cause)
}
