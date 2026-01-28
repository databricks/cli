package apps

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestIsIdempotencyError(t *testing.T) {
	t.Run("returns true when error contains keyword", func(t *testing.T) {
		err := errors.New("app is already in ACTIVE state")
		assert.True(t, isIdempotencyError(err, "ACTIVE state"))
	})

	t.Run("returns true when error contains any keyword", func(t *testing.T) {
		err := errors.New("already running")
		assert.True(t, isIdempotencyError(err, "ACTIVE state", "already"))
	})

	t.Run("returns false when error does not contain keywords", func(t *testing.T) {
		err := errors.New("something went wrong")
		assert.False(t, isIdempotencyError(err, "ACTIVE state", "already"))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, isIdempotencyError(nil, "ACTIVE state"))
	})

	t.Run("matches partial strings", func(t *testing.T) {
		err := errors.New("error: ACTIVE state detected")
		assert.True(t, isIdempotencyError(err, "ACTIVE state"))
	})
}

func TestFormatAppStatusMessage(t *testing.T) {
	t.Run("handles nil appInfo", func(t *testing.T) {
		msg := formatAppStatusMessage(nil, "test-app", "started")
		assert.Equal(t, "✔ App 'test-app' status: unknown", msg)
	})

	t.Run("handles unavailable app state", func(t *testing.T) {
		appInfo := &apps.App{
			AppStatus: &apps.ApplicationStatus{
				State: apps.ApplicationStateUnavailable,
			},
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateActive,
			},
		}
		msg := formatAppStatusMessage(appInfo, "test-app", "started")
		assert.Contains(t, msg, "unavailable")
		assert.Contains(t, msg, "ACTIVE")
	})

	t.Run("formats active state with 'is deployed' verb", func(t *testing.T) {
		appInfo := &apps.App{
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateActive,
			},
		}
		msg := formatAppStatusMessage(appInfo, "test-app", "is deployed")
		assert.Contains(t, msg, "already running")
		assert.Contains(t, msg, "ACTIVE")
	})

	t.Run("formats active state with 'started' verb", func(t *testing.T) {
		appInfo := &apps.App{
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateActive,
			},
		}
		msg := formatAppStatusMessage(appInfo, "test-app", "started")
		assert.Contains(t, msg, "started successfully")
		assert.Contains(t, msg, "ACTIVE")
	})

	t.Run("formats starting state", func(t *testing.T) {
		appInfo := &apps.App{
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateStarting,
			},
		}
		msg := formatAppStatusMessage(appInfo, "test-app", "started")
		assert.Contains(t, msg, "already starting")
		assert.Contains(t, msg, "STARTING")
	})

	t.Run("formats other compute states", func(t *testing.T) {
		appInfo := &apps.App{
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateStopped,
			},
		}
		msg := formatAppStatusMessage(appInfo, "test-app", "stopped")
		assert.Contains(t, msg, "status: STOPPED")
	})

	t.Run("handles nil compute status", func(t *testing.T) {
		appInfo := &apps.App{}
		msg := formatAppStatusMessage(appInfo, "test-app", "started")
		assert.Equal(t, "✔ App 'test-app' status: unknown", msg)
	})
}

func TestMakeArgsOptionalWithBundle(t *testing.T) {
	t.Run("updates command usage", func(t *testing.T) {
		cmd := &cobra.Command{}
		makeArgsOptionalWithBundle(cmd, "test [NAME]")
		assert.Equal(t, "test [NAME]", cmd.Use)
	})

	t.Run("sets Args validator", func(t *testing.T) {
		cmd := &cobra.Command{}
		makeArgsOptionalWithBundle(cmd, "test [NAME]")
		assert.NotNil(t, cmd.Args)
	})
}

func TestGetAppNameFromArgs(t *testing.T) {
	t.Run("returns arg when provided", func(t *testing.T) {
		cmd := &cobra.Command{}
		name, fromBundle, err := getAppNameFromArgs(cmd, []string{"my-app"})
		assert.NoError(t, err)
		assert.Equal(t, "my-app", name)
		assert.False(t, fromBundle)
	})
}

func TestUpdateCommandHelp(t *testing.T) {
	t.Run("sets Long help text", func(t *testing.T) {
		cmd := &cobra.Command{}
		updateCommandHelp(cmd, "Start", "start")
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("includes verb in help text", func(t *testing.T) {
		cmd := &cobra.Command{}
		updateCommandHelp(cmd, "Start", "start")
		assert.Contains(t, cmd.Long, "Start an app")
	})

	t.Run("includes command name in examples", func(t *testing.T) {
		cmd := &cobra.Command{}
		updateCommandHelp(cmd, "Stop", "stop")
		assert.Contains(t, cmd.Long, "databricks apps stop")
	})

	t.Run("includes all example scenarios", func(t *testing.T) {
		cmd := &cobra.Command{}
		updateCommandHelp(cmd, "Start", "start")
		assert.Contains(t, cmd.Long, "from a project directory")
		assert.Contains(t, cmd.Long, "--target prod")
		assert.Contains(t, cmd.Long, "my-app")
	})
}

func TestHandleAlreadyInStateError(t *testing.T) {
	t.Run("returns false when not an idempotency error", func(t *testing.T) {
		err := errors.New("some other error")
		cmd := &cobra.Command{}
		mockWrapper := func(cmd *cobra.Command, appName string, err error) error {
			return err
		}

		handled, _ := handleAlreadyInStateError(context.Background(), cmd, err, "test-app", []string{"ACTIVE"}, "is deployed", mockWrapper)
		assert.False(t, handled)
	})
}
