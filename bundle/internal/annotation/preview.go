package annotation

import (
	"sync"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal/clijson"
)

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

// launchStageOverrides maps a bundle config resource type to the launch stage
// its fields render at in the generated schema, regardless of the stage the
// upstream contract assigns each field. Resource types declare it through
// [resources.SchemaLaunchStageDeclarer]. Keys are the fully-qualified Go type
// paths (PkgPath + "." + Name), matching getPath in bundle/internal/schema.
var launchStageOverrides = sync.OnceValue(func() map[string]clijson.LaunchStage {
	res := map[string]clijson.LaunchStage{}
	for _, rt := range config.AllResourceTypes() {
		if d, ok := rt.New().(resources.SchemaLaunchStageDeclarer); ok {
			res[rt.Type.PkgPath()+"."+rt.Type.Name()] = d.SchemaLaunchStage()
		}
	}
	return res
})

// OverrideLaunchStage raises a field's launch stage to the type's configured
// override, unless the field's own stage is already more restrictive (see
// clijson.MoreRestrictive), in which case the field's stage is kept. This lets
// the CLI publish a resource at a chosen stage regardless of an upstream stage
// change, while still respecting a field the contract restricts further. A field
// of a type without a configured override is returned unchanged.
func OverrideLaunchStage(typePath string, stage clijson.LaunchStage) clijson.LaunchStage {
	override, ok := launchStageOverrides()[typePath]
	if !ok {
		return stage
	}
	if clijson.MoreRestrictive(stage, override) {
		return stage
	}
	return override
}
