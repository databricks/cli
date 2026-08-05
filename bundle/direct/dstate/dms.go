package dstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// overlayDMSState replaces the file-derived resource state with the state
// recorded in the deployment metadata service (DMS), when DMS owns this
// deployment. Once DMS is authoritative its resource set is trusted even when
// empty (a successful deploy with no resources); the file's resources are only
// used when DMS has no successful version, or when the user opts out of
// recording deployment history. The caller holds db.mu and has already
// populated db.Data from the file, including the DeploymentID.
//
// cfg is threaded in only for the temporary raw read in
// deploymentHasSuccessfulVersion; see the TODO there.
func (db *DeploymentState) overlayDMSState(ctx context.Context, client bundledeployments.BundleDeploymentsInterface, cfg *sdkconfig.Config) error {
	authoritative, err := deploymentHasSuccessfulVersion(ctx, cfg, db.Data.DeploymentID)
	if err != nil {
		return err
	}
	if !authoritative {
		// DMS has no completed version for this deployment: a prior direct deploy
		// that has not yet successfully recorded to DMS. Keep the file state.
		return nil
	}

	resources, err := fetchDeploymentResources(ctx, client, db.Data.DeploymentID)
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
//
// The deployment carries last_successful_version_id, which the server advances
// only when a version completes successfully (unlike last_version_id, which
// also advances on failure). So a non-empty value is exactly the "DMS owns the
// state" signal, readable in a single GetDeployment.
//
// TODO(DMS): this reads the deployment via a raw GET into a local struct
// because last_successful_version_id is still stage:DEVELOPMENT in the proto
// and therefore stripped from the generated SDK. Once the field is promoted to
// PRIVATE_PREVIEW and regenerated, replace the raw call with
// client.GetDeployment(...).LastSuccessfulVersionId and drop the cfg argument
// (revert overlayDMSState/Open back to taking only the typed client).
func deploymentHasSuccessfulVersion(ctx context.Context, cfg *sdkconfig.Config, deploymentID string) (bool, error) {
	apiClient, err := client.New(cfg)
	if err != nil {
		return false, fmt.Errorf("creating API client for deployment metadata service: %w", err)
	}

	// Mirrors the SDK's GetDeployment path (/api/2.0/bundle/{name} with
	// name=deployments/{id}); we unmarshal into a local struct so we can read
	// last_successful_version_id, which the typed SDK response drops.
	var dep struct {
		LastSuccessfulVersionID string `json:"last_successful_version_id"`
	}
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.0/bundle/deployments/"+deploymentID, auth.WorkspaceIDHeaders(cfg), nil, nil, &dep)
	if err != nil {
		// A deployment that was never recorded to DMS is not an error here: it
		// just means DMS is not (yet) the source of truth.
		if errors.Is(err, apierr.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("reading deployment from deployment metadata service: %w", err)
	}
	return dep.LastSuccessfulVersionID != "", nil
}

// fetchDeploymentResources lists every resource recorded for the deployment in
// DMS and maps them into state entries keyed by the fully-qualified resource key.
func fetchDeploymentResources(ctx context.Context, client bundledeployments.BundleDeploymentsInterface, deploymentID string) (map[string]ResourceEntry, error) {
	it := client.ListResources(ctx, bundledeployments.ListResourcesRequest{
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
