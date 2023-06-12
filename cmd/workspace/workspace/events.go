package workspace

type fileIOEvent struct {
	SourcePath string    `json:"source_path,omitempty"`
	TargetPath string    `json:"target_path,omitempty"`
	Type       EventType `json:"type"`
}

type EventType string

const (
	EventTypeFileExported    = EventType("FILE_EXPORTED")
	EventTypeExportStarted   = EventType("EXPORT_STARTED")
	EventTypeExportCompleted = EventType("EXPORT_COMPLETED")
	EventTypeFileSkipped     = EventType("FILE_SKIPPED")
)

func newFileExportedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeFileExported,
	}
}

func newExportCompletedEvent(targetPath string) fileIOEvent {
	return fileIOEvent{
		TargetPath: targetPath,
		Type:       EventTypeExportCompleted,
	}
}

func newFileSkippedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeFileSkipped,
	}
}

func newExportStartedEvent(sourcePath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		Type:       EventTypeExportStarted,
	}
}
