package workspace

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sync"
)

type fileIOEvent struct {
	SourcePath string `json:"source_path,omitempty"`
	TargetPath string `json:"target_path,omitempty"`
	Type       string `json:"type"`
}

func newImportStartedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       "IMPORT_STARTED",
	}
}

func newImportCompleteEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		Type: "IMPORT_COMPLETE",
	}
}

func newUploadCompleteEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       "UPLOAD_COMPLETE",
	}
}

func renderSyncEvents(ctx context.Context, eventChannel <-chan sync.Event, syncer *sync.Sync) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-eventChannel:
			if !ok {
				return nil
			}

			// We parse progress events from the sync to track when file uploads
			// are complete and emit the corresponding events
			if e.String() != "" && e.Type() == sync.EventTypeProgress {
				progressEvent := e.(*sync.EventSyncProgress)
				if progressEvent.Progress < 1 {
					return nil
				}
				// TODO: test this works with windows paths
				remotePath, err := syncer.RemotePath(progressEvent.Path)
				if err != nil {
					return err
				}
				err = cmdio.Render(ctx, newUploadCompleteEvent(progressEvent.Path, remotePath))
				if err != nil {
					return err
				}
			}
		}
	}
}
