package workspace

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sync"
)

// TODO: do not emit target directory in upload complete events.

type fileIOEvent struct {
	SourcePath string `json:"source_path,omitempty"`
	TargetPath string `json:"target_path,omitempty"`
	Type       string `json:"type"`
}

const (
	EventTypeImportStarted  = "IMPORT_STARTED"
	EventTypeImportComplete = "IMPORT_COMPLETE"
	EventTypeUploadComplete = "UPLOAD_COMPLETE"
)

func newImportStartedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeImportStarted,
	}
}

func newImportCompleteEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		Type:       EventTypeImportComplete,
		TargetPath: targetPath,
	}
}

func newUploadCompleteEvent(sourcePath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		Type:       EventTypeUploadComplete,
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
			if e.String() == "" {
				continue
			}
			switch v := e.(type) {
			case *sync.EventSyncProgress:
				// TODO: only emit this event if the the sync event has progress 1.o0
				// File upload has been completed. This renders the event for that
				// on the console
				err := cmdio.RenderWithTemplate(ctx, newUploadCompleteEvent(v.Path), "Uploaded {{.SourcePath}}\n")
				if err != nil {
					return err
				}
			}

		}
	}
}
