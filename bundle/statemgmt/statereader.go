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
	data, err := fetchDeploymentState(ctx, r.client, r.deploymentID)
	if err != nil {
		return err
	}
	db.OpenWithData(r.path, data)
	return nil
}

// fetchDeploymentState lists every resource recorded for the deployment and
// assembles them into a state Database.
func fetchDeploymentState(ctx context.Context, client sdkbundle.BundleInterface, deploymentID string) (dstate.Database, error) {
	resources, err := client.ListResourcesAll(ctx, sdkbundle.ListResourcesRequest{
		Parent: "deployments/" + deploymentID,
	})
	if err != nil {
		return dstate.Database{}, fmt.Errorf("listing resources from deployment metadata service: %w", err)
	}

	// Lineage and serial are file-state concepts for detecting concurrent
	// local edits; under DMS the server owns versioning, so they stay empty.
	data := dstate.NewDatabase("", 0)
	for _, res := range resources {
		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"), but the local state DB keys are fully qualified
		// ("resources.jobs.foo"), so prepend it here.
		key := "resources." + res.ResourceKey

		var state json.RawMessage
		if res.State != nil {
			state = *res.State
		}

		data.State[key] = dstate.ResourceEntry{
			ID:    res.ResourceId,
			State: state,
		}
	}
	return data, nil
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

	lineage, err := readLineage(path)
	if err != nil {
		return nil, err
	}
	if lineage == "" {
		return NewFileStateReader(path), nil
	}

	return NewDMSStateReader(b.WorkspaceClient(ctx).Bundle, lineage, path), nil
}

// readLineage reads the deployment lineage from the local resources.json state
// file. It returns an empty string when the file does not exist yet.
func readLineage(path string) (string, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var db dstate.Database
	if err := json.Unmarshal(content, &db); err != nil {
		return "", fmt.Errorf("parsing %s: %w", filepath.ToSlash(path), err)
	}
	return db.Lineage, nil
}
