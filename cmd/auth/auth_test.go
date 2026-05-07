package auth

import (
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
			wantErr:     `--profile "logfood" has host "https://logfood.cloud.databricks.com", which conflicts with --host "https://other.cloud.databricks.com". Use --profile only to select a profile`,
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

// TestProfileHostConflictViaCobra verifies that the conflict check runs
// through Cobra's lifecycle (PreRunE on login) and that the root command's
// PersistentPreRunE is NOT shadowed (it initializes logging, IO, user agent).
func TestProfileHostConflictViaCobra(t *testing.T) {
	// Point at a config file that has "profile-1" with host https://www.host1.com.
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")

	ctx := cmdctx.GenerateExecId(t.Context())
	cli := root.New(ctx)
	cli.AddCommand(New())

	// Set args: auth login --profile profile-1 --host https://other.host.com
	cli.SetArgs([]string{
		"auth", "login",
		"--profile", "profile-1",
		"--host", "https://other.host.com",
	})

	_, err := cli.ExecuteContextC(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `--profile "profile-1" has host "https://www.host1.com", which conflicts with --host "https://other.host.com"`)
	assert.Contains(t, err.Error(), "Use --profile only to select a profile")
}

// TestProfileHostConflictTokenViaCobra verifies the conflict check on the token subcommand.
func TestProfileHostConflictTokenViaCobra(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")

	ctx := cmdctx.GenerateExecId(t.Context())
	cli := root.New(ctx)
	cli.AddCommand(New())

	cli.SetArgs([]string{
		"auth", "token",
		"--profile", "profile-1",
		"--host", "https://other.host.com",
	})

	_, err := cli.ExecuteContextC(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `--profile "profile-1" has host "https://www.host1.com", which conflicts with --host "https://other.host.com"`)
}

// TestProfileHostCompatibleViaCobra verifies that matching --profile and --host
// pass the conflict check (the command will fail later for other reasons, but
// NOT with a conflict error).
func TestProfileHostCompatibleViaCobra(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")

	ctx := cmdctx.GenerateExecId(t.Context())
	cli := root.New(ctx)
	cli.AddCommand(New())

	cli.SetArgs([]string{
		"auth", "login",
		"--profile", "profile-1",
		"--host", "https://www.host1.com",
	})

	_, err := cli.ExecuteContextC(ctx)
	// The command may fail for other reasons (no browser, non-interactive, etc.)
	// but it should NOT fail with a conflict error.
	if err != nil {
		assert.NotContains(t, err.Error(), "conflicts with --host")
	}
}
