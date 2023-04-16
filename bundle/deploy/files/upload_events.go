package files

import "fmt"

type UploadEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (event *UploadEvent) String() string {
	return event.Message
}

func NewUploadStartedEvent() *UploadEvent {
	return &UploadEvent{
		Type:    "files_upload_started_event",
		Message: "Uploading bundle files",
	}
}

func NewUploadCompletedEvent(destination string) *UploadEvent {
	return &UploadEvent{
		Type:    "files_upload_completed_event",
		Message: fmt.Sprintf("Uploaded bundle files at %s\n", destination),
	}
}

func NewUploadFailedEvent() *UploadEvent {
	return &UploadEvent{
		Type:    "files_upload_failed_event",
		Message: "Failed to upload files",
	}
}
