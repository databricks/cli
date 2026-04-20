package apps

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleDeployOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	assert.NotNil(t, overrideFunc)

	cmd := &cobra.Command{}
	deployReq := &apps.CreateAppDeploymentRequest{}

	overrideFunc(cmd, deployReq)

	assert.Equal(t, "deploy [APP_NAME]", cmd.Use)
}

func TestBundleDeployOverrideFlags(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deployReq := &apps.CreateAppDeploymentRequest{}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deployReq)

	tests := []struct {
		name       string
		defaultVal string
	}{
		{"force", "false"},
		{"force-lock", "false"},
		{"fail-on-active-runs", "false"},
		{"compute-id", ""},
		{"cluster-id", ""},
		{"auto-approve", "false"},
		{"verbose", "false"},
		{"plan", ""},
		{"skip-validation", "false"},
		{"skip-tests", "true"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tc.name)
			require.NotNil(t, flag, "flag %q should be registered", tc.name)
			assert.Equal(t, tc.defaultVal, flag.DefValue)
		})
	}
}

func TestBundleDeployOverrideDeprecatedAndHiddenFlags(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deployReq := &apps.CreateAppDeploymentRequest{}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deployReq)

	computeID := cmd.Flags().Lookup("compute-id")
	require.NotNil(t, computeID)
	assert.NotEmpty(t, computeID.Deprecated, "compute-id should be deprecated")

	verbose := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verbose)
	assert.True(t, verbose.Hidden, "verbose should be hidden")
}

func TestBundleDeployOverrideClusterIDShorthand(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deployReq := &apps.CreateAppDeploymentRequest{}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deployReq)

	flag := cmd.Flags().Lookup("cluster-id")
	require.NotNil(t, flag)
	assert.Equal(t, "c", flag.Shorthand)
}

func TestBundleDeployOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deployReq := &apps.CreateAppDeploymentRequest{}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deployReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "app deployment")
	assert.Contains(t, cmd.Long, "project directory")
	assert.Contains(t, cmd.Long, "databricks.yml")
	assert.Contains(t, cmd.Long, "--auto-approve")
	assert.Contains(t, cmd.Long, "--force-lock")
}

func TestBundleDeployOverrideErrorWrapping(t *testing.T) {
	wrapperCalled := false
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		wrapperCalled = true
		assert.Equal(t, "test-app", appName)
		return err
	}

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("api error")
		},
	}
	deployReq := &apps.CreateAppDeploymentRequest{AppName: "test-app"}

	overrideFunc := BundleDeployOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deployReq)

	err := cmd.RunE(cmd, []string{"test-app"})
	assert.Error(t, err)
	assert.True(t, wrapperCalled)
}
