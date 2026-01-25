package apps

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleDeleteOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleDeleteOverrideWithWrapper(mockWrapper)
	assert.NotNil(t, overrideFunc)

	cmd := &cobra.Command{}
	deleteReq := &apps.DeleteAppRequest{}

	overrideFunc(cmd, deleteReq)

	assert.Equal(t, "delete [NAME]", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("auto-approve"))
	assert.NotNil(t, cmd.Flags().Lookup("force-lock"))
}

func TestBundleDeleteOverrideFlags(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deleteReq := &apps.DeleteAppRequest{}

	overrideFunc := BundleDeleteOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deleteReq)

	t.Run("auto-approve flag defaults to false", func(t *testing.T) {
		flag := cmd.Flags().Lookup("auto-approve")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("force-lock flag defaults to false", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force-lock")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestBundleDeleteOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	deleteReq := &apps.DeleteAppRequest{}

	overrideFunc := BundleDeleteOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deleteReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Delete an app")
	assert.Contains(t, cmd.Long, "project directory")
	assert.Contains(t, cmd.Long, "databricks.yml")
}

func TestBundleDeleteOverrideErrorWrapping(t *testing.T) {
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
	deleteReq := &apps.DeleteAppRequest{Name: "test-app"}

	overrideFunc := BundleDeleteOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, deleteReq)

	err := cmd.RunE(cmd, []string{"test-app"})
	assert.Error(t, err)
	assert.True(t, wrapperCalled)
}
