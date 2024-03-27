package mutator

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/build"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	currentVersion string
	minVersion     string
	maxVersion     string
	expectedError  string
}

func TestVerifyCliVersion(t *testing.T) {
	testCases := []testCase{
		{
			currentVersion: "0.0.1",
		},
		{
			currentVersion: "0.0.1",
			minVersion:     "0.100.0",
			expectedError:  "minimum Databricks CLI version required: v0.100.0, current version: v0.0.1",
		},
		{
			currentVersion: "0.100.0",
			minVersion:     "0.100.0",
		},
		{
			currentVersion: "0.100.1",
			minVersion:     "0.100.0",
		},
		{
			currentVersion: "0.100.0",
			maxVersion:     "1.0.0",
		},
		{
			currentVersion: "1.0.0",
			maxVersion:     "1.0.0",
		},
		{
			currentVersion: "1.0.0",
			maxVersion:     "0.100.0",
			expectedError:  "maximum Databricks CLI version required: v0.100.0, current version: v1.0.0",
		},
		{
			currentVersion: "0.99.0",
			minVersion:     "0.100.0",
			maxVersion:     "0.100.2",
			expectedError:  "minimum Databricks CLI version required: v0.100.0, current version: v0.99.0",
		},
		{
			currentVersion: "0.100.0",
			minVersion:     "0.100.0",
			maxVersion:     "0.100.2",
		},
		{
			currentVersion: "0.100.1",
			minVersion:     "0.100.0",
			maxVersion:     "0.100.2",
		},
		{
			currentVersion: "0.100.2",
			minVersion:     "0.100.0",
			maxVersion:     "0.100.2",
		},
		{
			currentVersion: "0.101.0",
			minVersion:     "0.100.0",
			maxVersion:     "0.100.2",
			expectedError:  "maximum Databricks CLI version required: v0.100.2, current version: v0.101.0",
		},
	}

	t.Cleanup(func() {
		build.SetBuildVersion(build.DefaultSemver)
	})

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testcase #%d", i), func(t *testing.T) {
			build.SetBuildVersion(tc.currentVersion)
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{
						MinDatabricksCliVersion: tc.minVersion,
						MaxDatabricksCliVersion: tc.maxVersion,
					},
				},
			}
			diags := bundle.Apply(context.Background(), b, VerifyCliVersion())
			if tc.expectedError != "" {
				require.NotEmpty(t, diags)
				require.Equal(t, tc.expectedError, diags.Error().Error())
			} else {
				require.Empty(t, diags)
			}
		})
	}
}
