package apps

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBundleDeleteOverrideWithWrapper(t *testing.T) {
	// Create a simple error wrapper for testing
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	// Create the override function
	overrideFunc := BundleDeleteOverrideWithWrapper(mockWrapper)
	assert.NotNil(t, overrideFunc, "BundleDeleteOverrideWithWrapper should return a non-nil function")

	// Create a test command
	cmd := &cobra.Command{}
	deleteReq := &apps.DeleteAppRequest{}

	// Apply the override
	overrideFunc(cmd, deleteReq)

	// Verify the command usage was updated
	assert.Equal(t, "delete [NAME]", cmd.Use, "Command usage should be updated to show optional NAME")

	// Verify flags were added
	assert.NotNil(t, cmd.Flags().Lookup("auto-approve"), "auto-approve flag should be added")
	assert.NotNil(t, cmd.Flags().Lookup("force-lock"), "force-lock flag should be added")
}
