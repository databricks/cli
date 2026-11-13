package dockercredentials_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
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

func recordedProfile(name string) profile.Profile {
	p := workspaceProfile(name)
	p.DockerCredentialHelper = true
	return p
}

func configSection(name, id string, recorded bool) string {
	section := fmt.Sprintf("[%s]\nhost = %s\nauth_type = databricks-cli\nworkspace_id = %s\n", name, workspaceHost, id)
	if recorded {
		section += "docker_credential_helper = true\n"
	}
	return section + "\n"
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
	m2m := profile.Profile{
		Name:                 "m2m",
		Host:                 workspaceHost,
		WorkspaceID:          workspaceID,
		HasClientCredentials: true,
	}
	recordedM2M := m2m
	recordedM2M.DockerCredentialHelper = true
	loopback := profile.Profile{
		Name:        "loopback",
		Host:        "http://127.0.0.1:8080",
		WorkspaceID: workspaceID,
		AuthType:    auth.AuthTypeDatabricksCli,
	}
	tests := []struct {
		name     string
		profiles profile.Profiles
		want     string
		wantErr  string
	}{
		{name: "selects workspace profile", profiles: profile.Profiles{workspaceProfile("workspace")}, want: "workspace"},
		{name: "ignores unsupported profile", profiles: profile.Profiles{m2m, workspaceProfile("workspace")}, want: "workspace"},
		{name: "ignores host the registry does not serve", profiles: profile.Profiles{loopback, workspaceProfile("workspace")}, want: "workspace"},
		{name: "selects recorded profile", profiles: profile.Profiles{workspaceProfile("dev"), recordedProfile("prod")}, want: "prod"},
		{
			name:     "no recorded profile",
			profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")},
			wantErr:  "multiple Databricks profiles match workspace ID 123456789: prod, dev. Select the profile for Docker with: databricks auth docker configure <profile-name>",
		},
		{
			name:     "recorded profile is unsupported",
			profiles: profile.Profiles{recordedM2M, workspaceProfile("workspace")},
			wantErr:  `profile "m2m" uses client credentials. Docker credential helper requires a profile created by databricks auth login`,
		},
		{
			name:     "several recorded profiles",
			profiles: profile.Profiles{recordedProfile("prod"), recordedProfile("dev")},
			wantErr:  "multiple Databricks profiles match workspace ID 123456789: prod, dev. Select the profile for Docker with: databricks auth docker configure <profile-name>",
		},
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
			got, err := dockercredentials.ProfileForRegistry(t.Context(), profile.InMemoryProfiler{Profiles: tt.profiles}, registry)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}

func TestRecordProfile(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: workspaceID,
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	tests := []struct {
		name         string
		config       string
		wantPrevious []string
		wantRecorded []string
		wantNoWrite  bool
	}{
		{
			name:         "records only profile",
			config:       configSection("prod", workspaceID, false),
			wantRecorded: []string{"prod"},
		},
		{
			name:         "switches from previous profiles",
			config:       configSection("dev", workspaceID, true) + configSection("prod", workspaceID, false) + configSection("stage", workspaceID, true),
			wantPrevious: []string{"dev", "stage"},
			wantRecorded: []string{"prod"},
		},
		{
			name:         "unmarks other profiles when already recorded",
			config:       configSection("dev", workspaceID, true) + configSection("prod", workspaceID, true),
			wantPrevious: []string{"dev"},
			wantRecorded: []string{"prod"},
		},
		{
			name:         "keeps recorded profile",
			config:       configSection("dev", workspaceID, false) + configSection("prod", workspaceID, true),
			wantRecorded: []string{"prod"},
			wantNoWrite:  true,
		},
		{
			name:         "ignores other workspaces",
			config:       configSection("other", "987654321", true) + configSection("prod", workspaceID, false),
			wantRecorded: []string{"other", "prod"},
		},
		{
			name:         "unmarks unsupported profiles",
			config:       "[pat]\nhost = " + workspaceHost + "\nauth_type = pat\nworkspace_id = " + workspaceID + "\ndocker_credential_helper = true\n\n" + configSection("prod", workspaceID, false),
			wantPrevious: []string{"pat"},
			wantRecorded: []string{"prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".databrickscfg")
			require.NoError(t, os.WriteFile(path, []byte(tt.config), 0o600))
			ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", path)

			previous, err := dockercredentials.RecordProfile(ctx, registry, "prod")
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrevious, previous)

			selected, err := dockercredentials.ProfileForRegistry(ctx, profile.DefaultProfiler, registry)
			require.NoError(t, err)
			assert.Equal(t, "prod", selected.Name)

			recorded, err := profile.DefaultProfiler.LoadProfiles(ctx, func(p profile.Profile) bool {
				return p.DockerCredentialHelper
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantRecorded, recorded.Names())
			if tt.wantNoWrite {
				contents, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, tt.config, string(contents))
			}
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
