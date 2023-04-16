package files

import "fmt"

type DeleteEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (event *DeleteEvent) String() string {
	return event.Message
}

func NewDeleteStartedEvent() *DeleteEvent {
	return &DeleteEvent{
		Type:    "files_delete_started_event",
		Message: "Starting deletion of remote bundle files",
	}
}

func NewDeleteRemoteDirInfoEvent(dir string) *DeleteEvent {
	return &DeleteEvent{
		Type:    "files_delete_remote_dir_info",
		Message: fmt.Sprintf("Bundle remote directory is %s", dir),
	}
}

func NewDeleteFailedEvent() *DeleteEvent {
	return &DeleteEvent{
		Type:    "files_delete_failed_event",
		Message: "failed to delete remote bundle files",
	}
}

func NewDeleteCompletedEvent() *DeleteEvent {
	return &DeleteEvent{
		Type:    "files_delete_completed_event",
		Message: "Successfully deleted files!",
	}
}

func NewDeletedSnapshotEvent(path string) *DeleteEvent {
	return &DeleteEvent{
		Type:    "snapshot_deleted_event",
		Message: fmt.Sprintf("Deleted snapshot file at %s", path),
	}
}
