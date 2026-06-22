package dstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go/apierr"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// overlayDMSState replaces the file-derived resource state with the state
// recorded in the deployment metadata service, when DMS owns this deployment.
// Once DMS is authoritative its resource set is trusted even when empty (a
// successful deploy with no resources); the file's resources are only used when
// DMS has no successful version, or when the user opts out of recording
// deployment history. The caller holds db.mu and has already populated db.Data
// from the file.
func (db *DeploymentState) overlayDMSState(ctx context.Context, client sdkbundle.BundleInterface) error {
	authoritative, err := deploymentHasSuccessfulVersion(ctx, client, db.Data.Lineage)
	if err != nil {
		return err
	}
	if !authoritative {
		// DMS has no completed version for this lineage: a prior direct deployment
		// that has not yet successfully recorded to DMS. Keep the file state.
		return nil
	}

	resources, err := fetchDeploymentResources(ctx, client, db.Data.Lineage)
	if err != nil {
		return err
	}

	db.Data.State = resources
	db.stateIDs = make(map[string]string, len(resources))
	for key, entry := range resources {
		db.stateIDs[key] = entry.ID
	}
	return nil
}

// deploymentHasSuccessfulVersion reports whether DMS holds a successfully
// completed version for the deployment. It is the signal that DMS owns the
// state: if the deployment was never recorded to DMS, or its initial DMS deploy
// did not complete successfully, DMS state is absent or partial and Open keeps
// the local file's resources instead.
func deploymentHasSuccessfulVersion(ctx context.Context, client sdkbundle.BundleInterface, deploymentID string) (bool, error) {
	// Versions are listed newest-first and fetched page by page, and we stop at
	// the first successful one, so a deployment with a long version history does
	// not require reading the whole list (typically just the first page).
	it := client.ListVersions(ctx, sdkbundle.ListVersionsRequest{
		Parent: "deployments/" + deploymentID,
	})
	for it.HasNext(ctx) {
		v, err := it.Next(ctx)
		if err != nil {
			// A deployment that was never recorded to DMS is not an error here: it just
			// means DMS is not (yet) the source of truth.
			if errors.Is(err, apierr.ErrNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("listing versions from deployment metadata service: %w", err)
		}
		if v.Status == sdkbundle.VersionStatusVersionStatusCompleted &&
			v.CompletionReason == sdkbundle.VersionCompleteVersionCompleteSuccess {
			return true, nil
		}
	}
	return false, nil
}

// fetchDeploymentResources lists every resource recorded for the deployment in
// DMS and maps them into state entries keyed by the fully-qualified resource key.
func fetchDeploymentResources(ctx context.Context, client sdkbundle.BundleInterface, deploymentID string) (map[string]ResourceEntry, error) {
	it := client.ListResources(ctx, sdkbundle.ListResourcesRequest{
		Parent: "deployments/" + deploymentID,
	})

	out := make(map[string]ResourceEntry)
	for it.HasNext(ctx) {
		res, err := it.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing resources from deployment metadata service: %w", err)
		}

		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"), but the state DB keys are fully qualified
		// ("resources.jobs.foo"), so prepend it here.
		key := "resources." + res.ResourceKey

		var state json.RawMessage
		if res.State != nil {
			state = *res.State
		}

		out[key] = ResourceEntry{
			ID:    res.ResourceId,
			State: state,
		}
	}
	return out, nil
}
