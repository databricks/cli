package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type prepareEnvironment struct{}

func (p *prepareEnvironment) Name() string {
	return "deploy:prepare-environment"
}

// Apply runs all pre-deployment environment preparation steps.
func (p *prepareEnvironment) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Check for resources in transient states and wait for them to complete.
	// This prevents deployment failures due to resources still being deleted.
	if err := waitForAppsDeletion(ctx, b); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// PrepareEnvironment returns a mutator that prepares the environment for deployment.
// It runs checks and waits for resources in transient states before deployment proceeds.
func PrepareEnvironment() bundle.Mutator {
	return &prepareEnvironment{}
}

// waitForAppsDeletion waits for apps to be deleted if they are in DELETING state.
func waitForAppsDeletion(ctx context.Context, b *bundle.Bundle) error {
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

		_, err := retries.Poll(ctx, 5*time.Minute, func() (*struct{}, *retries.Err) {
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
