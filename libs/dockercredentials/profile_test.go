package dockercredentials_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func TestProfilesForRegistry(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: workspaceID,
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	m2m := profile.Profile{
		Name:                 "m2m",
		Host:                 workspaceHost,
		WorkspaceID:          workspaceID,
		HasClientCredentials: true,
	}
	loopback := profile.Profile{
		Name:        "loopback",
		Host:        "http://127.0.0.1:8080",
		WorkspaceID: workspaceID,
		AuthType:    auth.AuthTypeDatabricksCli,
	}
	tests := []struct {
		name     string
		profiles profile.Profiles
		want     []string
		wantErr  string
	}{
		{name: "selects workspace profile", profiles: profile.Profiles{workspaceProfile("workspace")}, want: []string{"workspace"}},
		{
			name:     "shared workspace ID in file order",
			profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			want:     []string{"prod", "dev"},
		},
		{name: "ignores unsupported profile", profiles: profile.Profiles{m2m, workspaceProfile("workspace")}, want: []string{"workspace"}},
		{name: "ignores host the registry does not serve", profiles: profile.Profiles{loopback, workspaceProfile("workspace")}, want: []string{"workspace"}},
		{
			name:     "only unsupported profiles",
			profiles: profile.Profiles{m2m},
			wantErr:  `profile "m2m" uses client credentials. Docker credential helper requires a profile created by databricks auth login`,
		},
		{
			name:    "no matching profile",
			wantErr: "no Databricks profile found for workspace ID 123456789 from registry host 123456789.container.us-west-2.cloud.databricks.test. Run databricks auth login --host <workspace-url> and set workspace_id for that profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dockercredentials.ProfilesForRegistry(t.Context(), profile.InMemoryProfiler{Profiles: tt.profiles}, registry)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Names())
		})
	}
}

func TestTokenForRegistry(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: workspaceID,
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	errUnavailable := errors.New("service unavailable")
	tests := []struct {
		name      string
		profiles  profile.Profiles
		errs      map[string]error
		canceled  bool
		want      string
		wantTried []string
		wantErr   string
	}{
		{
			name:      "first logged-in profile wins",
			profiles:  profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			want:      "prod",
			wantTried: []string{"prod"},
		},
		{
			name:      "skips profiles that are logged out",
			profiles:  profile.Profiles{workspaceProfile("missing"), workspaceProfile("no-refresh"), workspaceProfile("revoked"), workspaceProfile("dev")},
			errs:      map[string]error{"missing": storage.ErrNotFound, "no-refresh": u2m.ErrMissingRefreshToken, "revoked": &u2m.InvalidRefreshTokenError{}},
			want:      "dev",
			wantTried: []string{"missing", "no-refresh", "revoked", "dev"},
		},
		{
			name:      "stops at other errors",
			profiles:  profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			errs:      map[string]error{"prod": errUnavailable},
			wantTried: []string{"prod"},
			wantErr:   `profile "prod": load token: service unavailable`,
		},
		{
			name:      "no logged-in profile",
			profiles:  profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			errs:      map[string]error{"prod": storage.ErrNotFound, "dev": storage.ErrNotFound},
			wantTried: []string{"prod", "dev"},
			wantErr:   "no logged-in Databricks profile for workspace ID 123456789 from registry host 123456789.container.us-west-2.cloud.databricks.test. Run databricks auth login --profile <profile-name> for one of these profiles:\nprofile \"prod\": load token: token not found\nprofile \"dev\": load token: token not found",
		},
		{
			name:     "stops when the context is canceled",
			profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			canceled: true,
			wantErr:  "context canceled",
		},
		{
			name:      "single profile keeps its error",
			profiles:  profile.Profiles{workspaceProfile("workspace")},
			errs:      map[string]error{"workspace": storage.ErrNotFound},
			wantTried: []string{"workspace"},
			wantErr:   "load token: token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tried []string
			load := func(_ context.Context, p profile.Profile) (*oauth2.Token, error) {
				tried = append(tried, p.Name)
				if err := tt.errs[p.Name]; err != nil {
					return nil, fmt.Errorf("load token: %w", err)
				}
				return &oauth2.Token{AccessToken: p.Name}, nil
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			got, err := dockercredentials.TokenForRegistry(ctx, profile.InMemoryProfiler{Profiles: tt.profiles}, registry, load)
			assert.Equal(t, tt.wantTried, tried)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.AccessToken)
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
