package paths

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
)

// artifactLocalPaths returns the local paths of artifacts, which are not a
// resource type. Their patterns start at the configuration root.
func artifactLocalPaths() []resources.LocalPath {
	// Base pattern to match all artifacts.
	base := dyn.NewPattern(
		dyn.Key("artifacts"),
		dyn.AnyKey(),
	)

	return []resources.LocalPath{
		{
			Pattern: base.Append(dyn.Key("path")),
			Mode:    resources.TranslateModeLocalAbsoluteDirectory,

			// Artifact paths may be outside the sync root.
			// They are the working directory for artifact builds.
			AllowOutsideSyncRoot: true,
		},
	}
}
