package apps

import (
	"testing"

	apps "github.com/databricks/sdk-go/apps/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBundleStopOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleStopOverrideWithWrapper[apps.StopAppRequest](mockWrapper)
	assert.NotNil(t, overrideFunc)

	cmd := &cobra.Command{}
	stopReq := &apps.StopAppRequest{}

	overrideFunc(cmd, stopReq)

	assert.Equal(t, "stop [NAME]", cmd.Use)
}

func TestBundleStopOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	stopReq := &apps.StopAppRequest{}

	overrideFunc := BundleStopOverrideWithWrapper[apps.StopAppRequest](mockWrapper)
	overrideFunc(cmd, stopReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Stop an app")
	assert.Contains(t, cmd.Long, "project directory")
}
