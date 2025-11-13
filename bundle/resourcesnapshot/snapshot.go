package resourcesnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

const snapshotFileName = "resource_snapshots.json"

// Snapshot stores the state of resources from the last successful deploy.
type Snapshot struct {
	Jobs      map[string]*jobs.Job                      `json:"jobs"`
	Pipelines map[string]*pipelines.GetPipelineResponse `json:"pipelines"`
}

func snapshotFilePath(b *bundle.Bundle) string {
	return path.Join(b.Config.Workspace.StatePath, snapshotFileName)
}

type save struct{}

func Save() bundle.Mutator {
	return &save{}
}

func (s *save) Name() string {
	return "resourcesnapshot.Save"
}

func (s *save) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	snapshot := &Snapshot{
		Jobs:      make(map[string]*jobs.Job),
		Pipelines: make(map[string]*pipelines.GetPipelineResponse),
	}

	w := b.WorkspaceClient()

	// Fetch and store job snapshots
	for key, job := range b.Config.Resources.Jobs {
		if job.ID == "" {
			log.Debugf(ctx, "Skipping job %s: no ID (not deployed)", key)
			continue
		}

		jobID, err := strconv.ParseInt(job.ID, 10, 64)
		if err != nil {
			log.Warnf(ctx, "Skipping job %s: invalid ID %q: %v", key, job.ID, err)
			continue
		}

		remoteJob, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
			JobId: jobID,
		})
		if err != nil {
			log.Warnf(ctx, "Failed to fetch job %s (ID: %d): %v", key, jobID, err)
			continue
		}

		snapshot.Jobs[key] = remoteJob
		log.Debugf(ctx, "Saved snapshot for job %s (ID: %d)", key, jobID)
	}

	// Fetch and store pipeline snapshots
	for key, pipeline := range b.Config.Resources.Pipelines {
		if pipeline.ID == "" {
			log.Debugf(ctx, "Skipping pipeline %s: no ID (not deployed)", key)
			continue
		}

		remotePipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{
			PipelineId: pipeline.ID,
		})
		if err != nil {
			log.Warnf(ctx, "Failed to fetch pipeline %s (ID: %s): %v", key, pipeline.ID, err)
			continue
		}

		snapshot.Pipelines[key] = remotePipeline
		log.Debugf(ctx, "Saved snapshot for pipeline %s (ID: %s)", key, pipeline.ID)
	}

	// Marshal snapshot to JSON
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return diag.FromErr(err)
	}

	// Write to WSFS
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return diag.FromErr(err)
	}

	err = f.Write(ctx, snapshotFileName, bytes.NewReader(data), filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Infof(ctx, "Saved resource snapshots to %s", snapshotFilePath(b))
	return nil
}

// Load reads the snapshot from WSFS.
// Returns nil if the snapshot file doesn't exist (e.g., first deploy).
func Load(ctx context.Context, b *bundle.Bundle) (*Snapshot, error) {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return nil, err
	}

	r, err := f.Read(ctx, snapshotFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debugf(ctx, "No previous snapshot found at %s", snapshotFilePath(b))
			return nil, nil
		}
		return nil, err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{}
	err = json.Unmarshal(data, snapshot)
	if err != nil {
		return nil, err
	}

	log.Debugf(ctx, "Loaded snapshot with %d jobs and %d pipelines from %s",
		len(snapshot.Jobs), len(snapshot.Pipelines), snapshotFilePath(b))

	return snapshot, nil
}
