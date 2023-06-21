package fs

type fileIOEvent struct {
	SourcePath string    `json:"source_path,omitempty"`
	TargetPath string    `json:"target_path,omitempty"`
	Type       EventType `json:"type"`
}

type EventType string

const (
	EventTypeFileCopied  = EventType("FILE_COPIED")
	EventTypeFileSkipped = EventType("FILE_SKIPPED")
)

func newFileCopiedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeFileCopied,
	}
}

func newFileSkippedEvent(sourcePath, targetPath string) fileIOEvent {
	return fileIOEvent{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Type:       EventTypeFileSkipped,
	}
}
