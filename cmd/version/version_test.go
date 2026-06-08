package version

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCheckTemplate(t *testing.T) {
	tests := []struct {
		name   string
		result versioncheck.Result
		want   string
	}{
		{
			name: "update available with upgrade command",
			result: versioncheck.Result{
				CurrentVersion:  "0.240.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: true,
				InstallMethod:   versioncheck.InstallHomebrew,
				UpgradeCommand:  "brew upgrade databricks",
			},
			want: `Databricks CLI v0.240.0
A new version is available: 0.245.0 (you have 0.240.0).
To upgrade, run:
  brew upgrade databricks
`,
		},
		{
			name: "update available without known install method",
			result: versioncheck.Result{
				CurrentVersion:  "0.240.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: true,
				InstallMethod:   versioncheck.InstallUnknown,
			},
			want: `Databricks CLI v0.240.0
A new version is available: 0.245.0 (you have 0.240.0).
Download the latest release: https://github.com/databricks/cli/releases
`,
		},
		{
			name: "up to date",
			result: versioncheck.Result{
				CurrentVersion:  "0.245.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: false,
			},
			want: `Databricks CLI v0.245.0
You're on the latest version.
`,
		},
		{
			name: "development build",
			result: versioncheck.Result{
				CurrentVersion:   "0.0.0-dev+abc123",
				DevelopmentBuild: true,
			},
			want: `Databricks CLI v0.0.0-dev+abc123
This is a development build; skipping the update check.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, out := cmdio.NewTestContextWithStdout(t.Context())
			err := cmdio.RenderWithTemplate(ctx, tt.result, "", updateCheckTemplate)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out.String())
		})
	}
}
