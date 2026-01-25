package apps

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBundleStartOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleStartOverrideWithWrapper(mockWrapper)
	assert.NotNil(t, overrideFunc, "BundleStartOverrideWithWrapper should return a non-nil function")

	cmd := &cobra.Command{}
	startReq := &apps.StartAppRequest{}

	overrideFunc(cmd, startReq)

	assert.Equal(t, "start [NAME]", cmd.Use, "Command usage should be updated to show optional NAME")
}

func TestBundleStopOverrideWithWrapper(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	overrideFunc := BundleStopOverrideWithWrapper(mockWrapper)
	assert.NotNil(t, overrideFunc)

	cmd := &cobra.Command{}
	stopReq := &apps.StopAppRequest{}

	overrideFunc(cmd, stopReq)

	assert.Equal(t, "stop [NAME]", cmd.Use)
}

func TestBundleStartOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	startReq := &apps.StartAppRequest{}

	overrideFunc := BundleStartOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, startReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Start an app")
	assert.Contains(t, cmd.Long, "project directory")
}

func TestBundleStopOverrideHelpText(t *testing.T) {
	mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
		return err
	}

	cmd := &cobra.Command{}
	stopReq := &apps.StopAppRequest{}

	overrideFunc := BundleStopOverrideWithWrapper(mockWrapper)
	overrideFunc(cmd, stopReq)

	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Stop an app")
	assert.Contains(t, cmd.Long, "project directory")
}
