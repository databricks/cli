package root

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const templateHelpersConfig = `
[spog]
host = https://spog.test
workspace_id = 111
token = token

[classic]
host = https://classic.test
workspace_id = 111
token = token

[other_workspace]
host = https://classic.test
workspace_id = 222
token = token

[no_workspace]
host = https://classic.test
workspace_id = none
token = token
`

func TestMustWorkspaceClientForTemplate(t *testing.T) {
	tests := []struct {
		name           string
		helpersProfile string
		wantErr        string
	}{
		{
			name: "unset",
		},
		{
			name:           "same workspace",
			helpersProfile: "classic",
		},
		{
			name:           "other workspace",
			helpersProfile: "other_workspace",
			wantErr:        `profile "other_workspace" from DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE must have the workspace ID of the selected profile, got "222" and "111"`,
		},
		{
			name:           "no workspace",
			helpersProfile: "no_workspace",
			wantErr:        `profile "no_workspace" from DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE must have the workspace ID of the selected profile, got "none" and "111"`,
		},
		{
			name:           "missing profile",
			helpersProfile: "missing",
			wantErr:        `failed to load profile "missing" from DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE: `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			configFile := filepath.Join(t.TempDir(), ".databrickscfg")
			require.NoError(t, os.WriteFile(configFile, []byte(templateHelpersConfig), 0o600))
			t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
			t.Setenv("DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE", tt.helpersProfile)

			ctx := SkipLoadBundle(cmdio.MockDiscard(t.Context()))
			cmd := New(ctx)
			require.NoError(t, cmd.Flag("profile").Value.Set("spog"))

			err := MustWorkspaceClientForTemplate(cmd, nil)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
