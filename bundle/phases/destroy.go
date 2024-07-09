package phases

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func assertRootPathExists(ctx context.Context, b *bundle.Bundle) (bool, error) {
	w := b.WorkspaceClient()
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.RootPath)

	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		log.Infof(ctx, "Root path does not exist: %s", b.Config.Workspace.RootPath)
		return false, nil
	}

	return true, err
}

// The destroy phase deletes artifacts and resources.
func Destroy() bundle.Mutator {
	destroyMutator := bundle.Seq(
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				terraform.StatePull(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.Plan(terraform.PlanGoal("destroy")),
				terraform.Destroy(),
				terraform.StatePush(),
				files.Delete(),
			),
			lock.Release(lock.GoalDestroy),
		),
		bundle.LogString("Destroy complete!"),
	)

	return newPhase(
		"destroy",
		[]bundle.Mutator{
			// Only run deploy mutator if root path exists.
			mutator.If(
				assertRootPathExists,
				destroyMutator,
				bundle.LogString("No active deployment found to destroy!"),
			),
		},
	)
}
