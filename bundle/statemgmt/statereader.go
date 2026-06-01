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

// StateReader populates the deployment resource-state DB used by the direct
// engine's plan/apply path. It abstracts where the state comes from: the local
// resources.json file (the historical source) or the deployment metadata
// service (DMS), which owns the state server-side when the bundle is opted into
// managed state.
//
// Both implementations leave the DB open in read mode. The plan path may later
// upgrade it to write mode; see dstate.DeploymentState.UpgradeToWrite.
type StateReader interface {
	// Load opens db and populates it with the deployment's resource state.
	Load(ctx context.Context, db *dstate.DeploymentState) error
}

// fileStateReader reads resource state from the local resources.json file.
type fileStateReader struct {
	path string
}

// NewFileStateReader returns a StateReader backed by the local resources.json
// file at path. This is the historical (non-DMS) source of resource state.
func NewFileStateReader(path string) StateReader {
	return &fileStateReader{path: path}
}

func (r *fileStateReader) Load(ctx context.Context, db *dstate.DeploymentState) error {
	// Recovery replays any leftover write-ahead log from a crashed deploy; the
	// file reader owns the on-disk state, so recovery applies here.
	return db.Open(ctx, r.path, dstate.WithRecovery(true), dstate.WithWrite(false))
}

// dmsStateReader reads resource state from the deployment metadata service.
type dmsStateReader struct {
	client       sdkbundle.BundleInterface
	deploymentID string

	// path is the local resources.json path. OpenWithData records it as the
	// eventual write target if the plan path later upgrades the DB to write
	// mode; the DMS reader itself never reads from or writes to it.
	path string
}

// NewDMSStateReader returns a StateReader backed by the deployment metadata
// service for the deployment identified by deploymentID, which must be
// non-empty. path is the local resources.json path (see dmsStateReader.path).
func NewDMSStateReader(client sdkbundle.BundleInterface, deploymentID, path string) StateReader {
	return &dmsStateReader{client: client, deploymentID: deploymentID, path: path}
}

func (r *dmsStateReader) Load(ctx context.Context, db *dstate.DeploymentState) error {
	// The deployment identity (lineage, serial) lives in the local resources.json:
	// the lineage IS the DMS deployment ID, and a deploy must keep using it rather
	// than minting a new one. Only the resource set is owned by the service, so we
	// preserve the local header and replace the resources with what DMS reports.
	data, err := readLocalDatabase(r.path)
	if err != nil {
		return err
	}

	resources, err := fetchDeploymentResources(ctx, r.client, r.deploymentID)
	if err != nil {
		return err
	}
	data.State = resources

	db.OpenWithData(r.path, data)
	return nil
}

// fetchDeploymentResources lists every resource recorded for the deployment and
// maps them into state entries keyed by the fully-qualified resource key.
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

// NewStateReader selects the StateReader for the bundle: the DMS reader when
// experimental.record_deployment_history is enabled and a prior deployment
// exists, otherwise the local resources.json reader.
//
// The DMS deployment ID is the state lineage, which is recorded in the local
// resources.json (see dstate.DeploymentState.GetOrInitLineage). When there is no
// lineage yet — a first deploy that has not registered with DMS — there is
// nothing to read from the service, so we read the (empty) local file instead.
func NewStateReader(ctx context.Context, b *bundle.Bundle, path string) (StateReader, error) {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return NewFileStateReader(path), nil
	}

	local, err := readLocalDatabase(path)
	if err != nil {
		return nil, err
	}
	// No lineage yet means no prior deployment registered with DMS, so there is
	// nothing to read from the service; fall back to the local file.
	if local.Lineage == "" {
		return NewFileStateReader(path), nil
	}

	return NewDMSStateReader(b.WorkspaceClient(ctx).Bundle, local.Lineage, path), nil
}

// readLocalDatabase parses the local resources.json state file. A missing file
// yields an empty database (no lineage), which callers treat as "no prior
// deployment".
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
