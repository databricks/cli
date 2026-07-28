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
// recorded in DMS, when DMS owns this deployment. An authoritative DMS is
// trusted even when its resource set is empty (a successful deploy of nothing).
// The caller holds db.mu and has already populated db.Data from the file.
func (db *DeploymentState) overlayDMSState(ctx context.Context, src *DMSSource) error {
	authoritative, err := deploymentHasSuccessfulVersion(ctx, src.Config, src.DeploymentID)
	if err != nil {
		return err
	}
	if !authoritative {
		// DMS has no completed version for this deployment: a prior direct deploy
		// that has not yet successfully recorded to DMS. Keep the file state.
		return nil
	}

	resources, err := fetchDeploymentResources(ctx, src.Client, src.DeploymentID, db.Data.State)
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

// deploymentHasSuccessfulVersion reports whether DMS owns the state. The server
// advances last_successful_version_id only when a version completes (unlike
// last_version_id, which also advances on failure), so a non-empty value means
// DMS holds a complete resource set. Otherwise Open keeps the file's resources.
//
// TODO(DMS): raw GET because last_successful_version_id is stage:DEVELOPMENT and
// stripped from the generated SDK. Once it ships, use
// client.GetDeployment(...).LastSuccessfulVersionId and drop DMSSource.Config.
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
//
// DMS has no field for dependency edges, and they cannot be recovered from the
// recorded state either: references are resolved to literals before it is
// serialized. So depends_on is carried over from the local state file.
//
// TODO(DMS): resources present in DMS but not in the local file therefore get no
// depends_on. Plan recomputes it from config for everything it still declares,
// so this only affects deletes of resources dropped from config, which are
// ordered arbitrarily among themselves.
func fetchDeploymentResources(ctx context.Context, client bundledeployments.BundleDeploymentsInterface, deploymentID string, local map[string]ResourceEntry) (map[string]ResourceEntry, error) {
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
			ID:        res.ResourceId,
			State:     state,
			DependsOn: local[key].DependsOn,
		}
	}
	return out, nil
}
