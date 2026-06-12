package annotation

// PreviewTag returns the human-readable launch-stage prefix to prepend to a
// field's or enum value's description. Others are skipped.
func PreviewTag(launchStage string) string {
	switch launchStage {
	case "PRIVATE_PREVIEW":
		return "[Private Preview]"
	case "PUBLIC_BETA":
		return "[Beta]"
	case "PUBLIC_PREVIEW":
		return "[Public Preview]"
	}
	return ""
}
