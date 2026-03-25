package auth

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProfileHostConflict(t *testing.T) {
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
			{Name: "staging", Host: "https://staging.cloud.databricks.com"},
			{Name: "no-host", Host: ""},
		},
	}

	cases := []struct {
		name        string
		profileName string
		host        string
		wantErr     string
	}{
		{
			name:        "matching hosts are allowed",
			profileName: "logfood",
			host:        "https://logfood.cloud.databricks.com",
		},
		{
			name:        "matching hosts with trailing slash",
			profileName: "logfood",
			host:        "https://logfood.cloud.databricks.com/",
		},
		{
			name:        "conflicting hosts produce error",
			profileName: "logfood",
			host:        "https://other.cloud.databricks.com",
			wantErr:     `--profile "logfood" has host "https://logfood.cloud.databricks.com", which conflicts with --host "https://other.cloud.databricks.com". Use --profile alone to select a profile`,
		},
		{
			name:        "profile not found skips check",
			profileName: "nonexistent",
			host:        "https://any.cloud.databricks.com",
		},
		{
			name:        "profile with no host skips check",
			profileName: "no-host",
			host:        "https://any.cloud.databricks.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			err := validateProfileHostConflict(ctx, tc.profileName, tc.host, profiler)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
