package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDatabricksCfg(t *testing.T) {
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}

	cfg := []byte("[PROFILE-1]\nhost = https://a.com\ntoken = a\n[PROFILE-2]\nhost = https://a.com\ntoken = b\n")
	err := os.WriteFile(filepath.Join(tempHomeDir, ".databrickscfg"), cfg, 0o644)
	assert.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv(homeEnvVar, tempHomeDir)
}

func emptyCommand(t *testing.T) *cobra.Command {
	ctx := context.Background()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	initProfileFlag(cmd)
	return cmd
}

func setupWithHost(t *testing.T, cmd *cobra.Command, host string) {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  host: %q
`, host)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)
}

func setupWithProfile(t *testing.T, cmd *cobra.Command, profile string) {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  profile: %q
`, profile)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)
}

func TestBundleConfigureProfile(t *testing.T) {
	tcases := []struct {
		name         string
		hostInConfig string

		// --profile flag
		profileFlag string
		// DATABRICKS_CONFIG_PROFILE environment variable
		profileEnvVar string
		// profile in config
		profileInConfig string

		expectedError   string
		expectedHost    string
		expectedProfile string
		expectedToken   string
	}{
		{
			name:         "no match, keep host",
			hostInConfig: "https://x.com",

			expectedHost: "https://x.com",
		},
		{
			name:         "multiple profile matches",
			hostInConfig: "https://a.com",

			expectedError: "multiple profiles matched: PROFILE-1, PROFILE-2",
		},
		{
			name:         "non-existent profile",
			profileFlag:  "NOEXIST",
			hostInConfig: "https://x.com",

			expectedError: "has no NOEXIST profile configured",
		},
		{
			name:         "mismatched profile",
			hostInConfig: "https://x.com",
			profileFlag:  "PROFILE-1",

			expectedError: "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com",
		},
		{
			name:         "profile flag specified",
			hostInConfig: "https://a.com",
			profileFlag:  "PROFILE-1",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-1",
		},
		{
			name:          "mismatched profile env variable",
			hostInConfig:  "https://x.com",
			profileEnvVar: "PROFILE-1",

			expectedError: "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com",
		},
		{
			// The --profile flag takes precedence over the DATABRICKS_CONFIG_PROFILE environment variable
			name:          "(host) profile flag takes precedence over env variable",
			hostInConfig:  "https://a.com",
			profileFlag:   "PROFILE-1",
			profileEnvVar: "NOEXIST",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-1",
		},
		{
			name:            "profile from config",
			profileInConfig: "PROFILE-1",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-1",
			expectedToken:   "a",
		},
		{
			// The --profile flag takes precedence over the profile in the databricks.yml file
			name:            "profile flag takes precedence",
			profileInConfig: "PROFILE-1",
			profileFlag:     "PROFILE-2",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-2",
			expectedToken:   "b",
		},
		{
			// The DATABRICKS_CONFIG_PROFILE environment variable takes precedence over the profile in the databricks.yml file
			name:            "profile env variable takes precedence",
			profileInConfig: "PROFILE-1",
			profileEnvVar:   "PROFILE-2",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-2",
			expectedToken:   "b",
		},
		{
			// The --profile flag takes precedence over the DATABRICKS_CONFIG_PROFILE environment variable
			name:            "profile flag takes precedence over env variable",
			profileInConfig: "PROFILE-1",
			profileFlag:     "PROFILE-2",
			profileEnvVar:   "NOEXIST",

			expectedHost:    "https://a.com",
			expectedProfile: "PROFILE-2",
			expectedToken:   "b",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)

			cmd := emptyCommand(t)

			// Set up host in databricks.yml
			if tc.hostInConfig != "" {
				setupWithHost(t, cmd, tc.hostInConfig)
			}

			// Set up profile in databricks.yml
			if tc.profileInConfig != "" {
				setupWithProfile(t, cmd, tc.profileInConfig)
			}

			// Set --profile flag
			if tc.profileFlag != "" {
				err := cmd.Flag("profile").Value.Set(tc.profileFlag)
				require.NoError(t, err)
			}

			// Set DATABRICKS_CONFIG_PROFILE environment variable
			if tc.profileEnvVar != "" {
				t.Setenv("DATABRICKS_CONFIG_PROFILE", tc.profileEnvVar)
			}

			_, diags := MustConfigureBundle(cmd)

			if tc.expectedError != "" {
				assert.ErrorContains(t, diags.Error(), tc.expectedError)
			} else {
				assert.NoError(t, diags.Error())
			}

			// Assert on the resolved configuration values
			if tc.expectedHost != "" {
				assert.Equal(t, tc.expectedHost, ConfigUsed(cmd.Context()).Host)
			}
			if tc.expectedProfile != "" {
				assert.Equal(t, tc.expectedProfile, ConfigUsed(cmd.Context()).Profile)
			}
			if tc.expectedToken != "" {
				assert.Equal(t, tc.expectedToken, ConfigUsed(cmd.Context()).Token)
			}
		})
	}
}

func TestTargetFlagFull(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "--target", "development"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}

func TestTargetFlagShort(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "-t", "production"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "production", getTarget(cmd))
}

// TODO: remove when environment flag is fully deprecated
func TestTargetEnvironmentFlag(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	initEnvironmentFlag(cmd)
	cmd.SetArgs([]string{"version", "--environment", "development"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}
