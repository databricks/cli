package annotation

// PreviewTag returns the human-readable launch-stage prefix to prepend to a
// field's description. It is the single source of truth shared by the schema
// and docs generators. cli.json is filtered at min-stage=PRIVATE_PREVIEW
// upstream, so DEVELOPMENT never reaches here and GA yields no tag.
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
