package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
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
	// Point at a config file that has "profile-1" with host https://www.host1.test.
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")

	ctx := cmdctx.GenerateExecId(t.Context())
	cli := root.New(ctx)
	cli.AddCommand(New())

	// Set args: auth login --profile profile-1 --host https://other.host.test
	cli.SetArgs([]string{
		"auth", "login",
		"--profile", "profile-1",
		"--host", "https://other.host.test",
	})

	_, err := cli.ExecuteContextC(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `--profile "profile-1" has host "https://www.host1.test", which conflicts with --host "https://other.host.test"`)
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
		"--host", "https://other.host.test",
	})

	_, err := cli.ExecuteContextC(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `--profile "profile-1" has host "https://www.host1.test", which conflicts with --host "https://other.host.test"`)
}

// Matching flags reach the stubbed run boundary after the real conflict check.
func TestProfileHostCompatibleViaCobra(t *testing.T) {
	// Keep Cobra's profile fixture in the owned test directory.
	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	fixture, err := os.ReadFile("./testdata/.databrickscfg")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, fixture, 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	ctx := cmdctx.GenerateExecId(t.Context())
	cli := root.New(ctx)
	cli.AddCommand(New())
	login, _, err := cli.Find([]string{"auth", "login"})
	require.NoError(t, err)
	reachedRun := false
	login.RunE = func(*cobra.Command, []string) error { reachedRun = true; return nil }

	cli.SetArgs([]string{
		"auth", "login",
		"--profile", "profile-1",
		"--host", "https://www.host1.test",
	})

	_, err = cli.ExecuteContextC(ctx)
	assert.True(t, reachedRun)
	if err != nil {
		assert.NotContains(t, err.Error(), "conflicts with --host")
	}
}

func TestPromptForHostRejectsEmptyHost(t *testing.T) {
	tm := termtest.Run(t, promptForHost)
	tm.WaitFor("Databricks host")
	tm.Type(termtest.KeyEnter)
	tm.Golden("01-empty")
	tm.Type("  " + termtest.KeyEnter)
	tm.Golden("02-spaces")
	tm.Type("https://workspace.test " + termtest.KeyEnter)
	host, err := tm.Result()
	require.NoError(t, err)
	assert.Equal(t, "https://workspace.test", host)
}

func TestPromptForHostRejectsHostEmptyAfterNormalization(t *testing.T) {
	tm := termtest.Run(t, promptForHost)
	tm.WaitFor("Databricks host")
	tm.Type("?o=123" + termtest.KeyEnter)
	tm.Golden("01-query-params")
	tm.Type(strings.Repeat(termtest.KeyBackspace, len("?o=123")) + "https://workspace.test/?o=123" + termtest.KeyEnter)
	host, err := tm.Result()
	require.NoError(t, err)
	assert.Equal(t, "https://workspace.test/?o=123", host)
}
