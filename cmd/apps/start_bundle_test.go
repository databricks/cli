package apps

import (
	"testing"

	apps "github.com/databricks/sdk-go/apps/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBundleStartOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleStartOverrideWithWrapper[apps.StartAppRequest](mockWrapper)
	assert.NotNil(t, overrideFunc)

	cmd := &cobra.Command{}
	startReq := &apps.StartAppRequest{}

	overrideFunc(cmd, startReq)

	assert.Equal(t, "start [NAME]", cmd.Use)
}

func TestBundleStartOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	startReq := &apps.StartAppRequest{}

	overrideFunc := BundleStartOverrideWithWrapper[apps.StartAppRequest](mockWrapper)
	overrideFunc(cmd, startReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Start an app")
	assert.Contains(t, cmd.Long, "project directory")
}
