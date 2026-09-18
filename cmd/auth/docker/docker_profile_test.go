package docker

import (
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
)

type invalidRefreshTokenTestError struct {
	error
}

func (e *invalidRefreshTokenTestError) As(target any) bool {
	invalidRefreshToken, ok := target.(**u2m.InvalidRefreshTokenError)
	if !ok {
		return false
	}
	*invalidRefreshToken = new(u2m.InvalidRefreshTokenError)
	return true
}

func TestValidateDockerCredentialProfile(t *testing.T) {
	tests := []struct {
		name      string
		profile   profile.Profile
		wantError string
	}{
		{
			name: "workspace",
			profile: profile.Profile{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: "123456789",
				AuthType:    auth.AuthTypeDatabricksCli,
			},
		},
		{
			name: "client credentials",
			profile: profile.Profile{
				Name:                 "m2m",
				Host:                 "https://workspace.cloud.databricks.test",
				WorkspaceID:          "123456789",
				HasClientCredentials: true,
			},
			wantError: "requires a profile created by databricks auth login",
		},
		{
			name: "unsupported auth type",
			profile: profile.Profile{
				Name:        "pat",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: "123456789",
				AuthType:    "pat",
			},
			wantError: "requires a profile created by databricks auth login",
		},
		{
			name: "classic account",
			profile: profile.Profile{
				Name:        "account",
				Host:        "https://accounts.cloud.databricks.test",
				AccountID:   "account-id",
				WorkspaceID: "123456789",
				AuthType:    auth.AuthTypeDatabricksCli,
			},
			wantError: "does not target a workspace",
		},
		{
			name: "unified account",
			profile: profile.Profile{
				Name:      "account",
				Host:      "https://workspace.cloud.databricks.test",
				AccountID: "account-id",
				AuthType:  auth.AuthTypeDatabricksCli,
			},
			wantError: "does not target a workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDockerCredentialProfile(tt.profile)
			if tt.wantError == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}
