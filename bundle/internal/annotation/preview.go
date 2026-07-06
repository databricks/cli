package annotation

import "github.com/databricks/cli/internal/clijson"

// previewTags maps each launch stage to the human-readable prefix prepended to
// a field's or enum value's description. clijson owns the closed set of stages;
// this map must cover every one (asserted by TestPreviewTagCoversAllStages). A
// stage mapping to "" (GA) renders no prefix.
var previewTags = map[clijson.LaunchStage]string{
	clijson.LaunchStageGA:             "",
	clijson.LaunchStagePublicPreview:  "[Public Preview]",
	clijson.LaunchStagePublicBeta:     "[Beta]",
	clijson.LaunchStagePrivatePreview: "[Private Preview]",
}

// PreviewTag returns the human-readable launch-stage prefix to prepend to a
// field's or enum value's description. GA and the empty stage return "".
func PreviewTag(stage clijson.LaunchStage) string {
	return previewTags[stage]
}
