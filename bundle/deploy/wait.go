package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// WaitForAppDeletion waits for apps to be deleted if they are in DELETING state.
func WaitForAppDeletion(ctx context.Context, b *bundle.Bundle) error {
	if len(b.Config.Resources.Apps) == 0 {
		return nil
	}

	w := b.WorkspaceClient()

	for _, app := range b.Config.Resources.Apps {
		appName := app.Name
		if appName == "" {
			continue
		}

		log.Debugf(ctx, "Checking status of app %s", appName)

		_, err := retries.Poll(ctx, 10*time.Minute, func() (*struct{}, *retries.Err) {
			appStatus, err := w.Apps.GetByName(ctx, appName)
			if err != nil {
				if apierr.IsMissing(err) {
					return nil, nil
				}
				return nil, retries.Halt(err)
			}

			if appStatus.ComputeStatus.State == apps.ComputeStateDeleting {
				log.Infof(ctx, "App %s is in DELETING state, waiting for it to be deleted...", appName)
				return nil, retries.Continues("app is deleting")
			}

			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("failed to wait for app %s deletion: %w", appName, err)
		}
	}

	return nil
}
