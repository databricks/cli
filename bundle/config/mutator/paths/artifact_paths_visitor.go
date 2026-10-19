package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitArtifactPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Artifacts) {
		if a := root.Artifacts[name]; a != nil {
			if err := visit(fn, visitArgs{
				owner: a,
				field: "path",
				value: &a.Path,
				mode:  TranslateModeLocalAbsoluteDirectory,
				path:  func() *structpath.PathNode { return structpath.NewPath(nil, "artifacts", name, "path") },
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
