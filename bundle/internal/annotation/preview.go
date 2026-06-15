package annotation

import "fmt"

// LaunchStage is a field, enum value, or type's release stage as carried by the
// cli.json contract's launch_stage. Only the constants below are recognized;
// ParseLaunchStage rejects any other value so a stage introduced upstream can't
// slip through unmapped.
type LaunchStage string

const (
	LaunchStageGA             LaunchStage = "GA"
	LaunchStagePublicPreview  LaunchStage = "PUBLIC_PREVIEW"
	LaunchStagePublicBeta     LaunchStage = "PUBLIC_BETA"
	LaunchStagePrivatePreview LaunchStage = "PRIVATE_PREVIEW"
)

// previewTags is the single source of truth for the launch stages the CLI
// recognizes and the human-readable prefix each contributes to a field's or
// enum value's description. A stage mapping to "" (GA) renders no prefix.
// ParseLaunchStage validates against these keys, so adding a stage upstream
// without extending this map fails codegen.
var previewTags = map[LaunchStage]string{
	LaunchStageGA:             "",
	LaunchStagePublicPreview:  "[Public Preview]",
	LaunchStagePublicBeta:     "[Beta]",
	LaunchStagePrivatePreview: "[Private Preview]",
}

// ParseLaunchStage converts a raw launch_stage string from the contract into a
// LaunchStage, mapping the empty string to GA (the implicit default). It errors
// on any value the CLI doesn't recognize, forcing a human to extend previewTags
// when upstream introduces a stage.
func ParseLaunchStage(s string) (LaunchStage, error) {
	if s == "" {
		return LaunchStageGA, nil
	}
	stage := LaunchStage(s)
	if _, ok := previewTags[stage]; !ok {
		return "", fmt.Errorf("unknown launch stage %q: add it to previewTags in bundle/internal/annotation/preview.go", s)
	}
	return stage, nil
}

// PreviewTag returns the human-readable launch-stage prefix to prepend to a
// field's or enum value's description. GA and the empty stage return "".
func PreviewTag(stage LaunchStage) string {
	return previewTags[stage]
}
