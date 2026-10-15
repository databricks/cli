package apps

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
)

// waitForAppCompute polls the app until its compute reaches target, calling
// callback with every polled state. Reaching ERROR, or STOPPED when waiting
// for ACTIVE, fails the wait. sdk-go waiters report no progress, so the
// commands that show a spinner poll here instead.
func waitForAppCompute(ctx context.Context, w *sdkclients.Clients, name string, target sdkapps.ComputeStatus_ComputeState, timeout time.Duration, callback func(*sdkapps.App)) (*sdkapps.App, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastStatus string
	for attempt := 1; ; attempt++ {
		app, err := w.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(name)})
		if err != nil {
			if ctx.Err() != nil && lastStatus != "" {
				return nil, fmt.Errorf("timed out after %s: %s", timeout, lastStatus)
			}
			return nil, err
		}
		if callback != nil {
			callback(app)
		}
		var status sdkapps.ComputeStatus_ComputeState
		if app.ComputeStatus != nil {
			status = app.ComputeStatus.State
		}
		statusMessage := fmt.Sprintf("current status: %s", status)
		if app.ComputeStatus != nil {
			statusMessage = utils.Deref(app.ComputeStatus.Message)
		}
		switch {
		case status == target:
			return app, nil
		case status == sdkapps.ComputeStatus_ComputeState_Error,
			status == sdkapps.ComputeStatus_ComputeState_Stopped && target == sdkapps.ComputeStatus_ComputeState_Active:
			return nil, fmt.Errorf("failed to reach %s, got %s: %s", target, status, statusMessage)
		}
		lastStatus = statusMessage

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out after %s: %s", timeout, lastStatus)
		case <-time.After(min(time.Duration(attempt)*time.Second, 10*time.Second)):
		}
	}
}
