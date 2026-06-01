package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dstate"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// StateReader loads a deployment's state into the in-memory DeploymentState that
// the direct engine reads during plan/apply.
//
// Deployment state has two parts:
//
//   - Identity: the lineage (the deployment id) and serial. These always live in
//     the local resources.json file and identify the deployment across runs.
//   - Resources: each deployed resource's id and last-deployed config.
//
// Where the resources come from depends on whether the bundle records deployment
// history, which is what the two implementations capture:
//
//   - fileStateReader: identity and resources both come from resources.json.
//     This is the default and matches the behavior before DMS.
//   - dmsStateReader: identity still comes from resources.json (its lineage is the
//     DMS deployment id), but the resource set is read from the deployment
//     metadata service (DMS).
//
// Both leave the DB open for reading; the plan path may later upgrade it to write
// mode (see dstate.DeploymentState.UpgradeToWrite).
type StateReader interface {
	// Load populates db with the deployment's state, leaving it open for reading.
	Load(ctx context.Context, db *dstate.DeploymentState) error
}

// fileStateReader loads both identity and resources from the local resources.json
// file. "local" means the file in the bundle's state cache that PullResourcesState
// has already synced from the workspace.
type fileStateReader struct {
	path string
}

// NewFileStateReader returns a StateReader that reads both identity and resources
// from the local resources.json file at path.
func NewFileStateReader(path string) StateReader {
	return &fileStateReader{path: path}
}

func (r *fileStateReader) Load(ctx context.Context, db *dstate.DeploymentState) error {
	// Open reads resources.json (identity + resources) into db. Recovery replays a
	// leftover write-ahead log from a crashed deploy, which only the file-backed
	// state can have.
	return db.Open(ctx, r.path, dstate.WithRecovery(true), dstate.WithWrite(false))
}

// dmsStateReader loads the identity from the local resources.json file and the
// resource set from DMS. If DMS has no resources for the deployment yet — an
// existing file-based deployment that just enabled record_deployment_history —
// it keeps the resources from resources.json so they are not re-created; the next
// deploy records them to DMS.
type dmsStateReader struct {
	client       sdkbundle.BundleInterface
	deploymentID string
	path         string // local resources.json path; supplies the identity (lineage/serial)
}

// NewDMSStateReader returns a StateReader that reads identity from the local
// resources.json at path and the resource set from DMS for deploymentID (which is
// the deployment's lineage).
func NewDMSStateReader(client sdkbundle.BundleInterface, deploymentID, path string) StateReader {
	return &dmsStateReader{client: client, deploymentID: deploymentID, path: path}
}

func (r *dmsStateReader) Load(ctx context.Context, db *dstate.DeploymentState) error {
	// Identity (lineage/serial) always comes from resources.json: the lineage is
	// the DMS deployment id and a later deploy must reuse it rather than mint a new
	// one. resources.json keeps recording resources too, so the bundle can be
	// migrated back to file-based state.
	data, err := readLocalDatabase(r.path)
	if err != nil {
		return err
	}

	resources, err := fetchDeploymentResources(ctx, r.client, r.deploymentID)
	if err != nil {
		return err
	}

	// Once DMS has the resources, they are the source of truth. Until then — e.g.
	// just after enabling record_deployment_history on an existing file-based
	// deployment — DMS returns nothing, so keep resources.json's resources rather
	// than treating everything as new. The next deploy records them to DMS.
	if len(resources) > 0 {
		data.State = resources
	}

	db.OpenWithData(r.path, data)
	return nil
}

// fetchDeploymentResources lists every resource recorded for the deployment in
// DMS and maps them into state entries keyed by the fully-qualified resource key.
func fetchDeploymentResources(ctx context.Context, client sdkbundle.BundleInterface, deploymentID string) (map[string]dstate.ResourceEntry, error) {
	resources, err := client.ListResourcesAll(ctx, sdkbundle.ListResourcesRequest{
		Parent: "deployments/" + deploymentID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing resources from deployment metadata service: %w", err)
	}

	out := make(map[string]dstate.ResourceEntry, len(resources))
	for _, res := range resources {
		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"), but the local state DB keys are fully qualified
		// ("resources.jobs.foo"), so prepend it here.
		key := "resources." + res.ResourceKey

		var state json.RawMessage
		if res.State != nil {
			state = *res.State
		}

		out[key] = dstate.ResourceEntry{
			ID:    res.ResourceId,
			State: state,
		}
	}
	return out, nil
}

// NewStateReader picks the reader for the bundle. Three cases:
//
//  1. record_deployment_history off: read everything from resources.json.
//  2. on, but resources.json has no lineage (nothing deployed yet): there is no
//     DMS deployment to read, so read the (empty) local file.
//  3. on, with a lineage: read resources from DMS for that deployment id. If DMS
//     has none yet (record_deployment_history was just enabled on an existing
//     deployment), the DMS reader keeps resources.json's resources; see
//     dmsStateReader.Load.
//
// The lineage comes from the local resources.json, so PullResourcesState must
// have synced it before this is called.
func NewStateReader(ctx context.Context, b *bundle.Bundle, path string) (StateReader, error) {
	recordHistory := b.Config.Experimental != nil && b.Config.Experimental.RecordDeploymentHistory
	if !recordHistory {
		return NewFileStateReader(path), nil // case 1
	}

	local, err := readLocalDatabase(path)
	if err != nil {
		return nil, err
	}
	if local.Lineage == "" {
		return NewFileStateReader(path), nil // case 2
	}

	return NewDMSStateReader(b.WorkspaceClient(ctx).Bundle, local.Lineage, path), nil // case 3
}

// readLocalDatabase parses the local resources.json file. A missing file yields
// an empty database (no lineage), which callers read as "nothing deployed yet".
func readLocalDatabase(path string) (dstate.Database, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return dstate.NewDatabase("", 0), nil
	}
	if err != nil {
		return dstate.Database{}, err
	}

	var db dstate.Database
	if err := json.Unmarshal(content, &db); err != nil {
		return dstate.Database{}, fmt.Errorf("parsing %s: %w", filepath.ToSlash(path), err)
	}
	return db, nil
}
