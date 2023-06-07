package workspace

type fileIOEvent struct {
	SourcePath string `json:"source_path,omitempty"`
	TargetPath string `json:"target_path,omitempty"`
	Type       string `json:"type"`
}

const EventTypeDownloadComplete = "DOWNLOAD_COMPLETE"
const EventTypeExportStarted = "EXPORT_STARTED"
const EventTypeExportCompleted = "EXPORT_COMPLETED"
const EventTypeFileSkipped = "FILE_SKIPPED"

func newDownloadCompleteEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeDownloadComplete,
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
