package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitGenieSpacePaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.GenieSpaces) {
		if g := root.Resources.GenieSpaces[name]; g != nil {
			if err := visit(fn, visitArgs{
				owner: g,
				field: "file_path",
				value: &g.FilePath,
				mode:  TranslateModeLocalRelative,
				path: func() *structpath.PathNode {
					return structpath.NewPath(resourcePath("genie_spaces", name), "file_path")
				},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
