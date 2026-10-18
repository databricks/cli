package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitArtifactPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Artifacts) {
		if a := root.Artifacts[name]; a != nil {
			if err := visit(&a.Path, TranslateModeLocalAbsoluteDirectory, nil, fn, func() *structpath.PathNode { return structpath.NewPath(nil, "artifacts", name, "path") }); err != nil {
				return err
			}
		}
	}
	return nil
}
