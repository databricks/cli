package protos

// FastFeedbackLog corresponds to the FastFeedbackLog lumberjack proto in universe
// (proto/logs/frontend/docs-platform/fast_feedback_log.proto). It is the event the
// in-product feedback widget records, so CLI feedback is triaged the same way.
type FastFeedbackLog struct {
	// ID of an event the feedback relates to. The CLI sets the command execution ID so
	// triagers can find the matching requests in http_access_log.
	ReferenceEventID string `json:"reference_event_id,omitempty"`

	// ES component the feedback is filed under (e.g. "DeveloperEcosystem.DatabricksCLI").
	ParentUI string `json:"parent_ui,omitempty"`

	// Feature within the parent UI the feedback is about.
	Origin string `json:"origin,omitempty"`

	CanBeContacted bool `json:"can_be_contacted"`

	// Comma-separated "key:value" pairs of component versions.
	UIVersions string `json:"ui_versions,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	// Feedback text written by the user.
	Text string `json:"text,omitempty"`
}
