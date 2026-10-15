package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func VisitGenieSpacePaths(root structvar.View, fn VisitFunc) error {
	pattern := structpath.NewPattern(nil, "resources", "genie_spaces", structpath.AnyKey, "file_path")

	return structvar.ForEach(root, pattern, func(path *structpath.PathNode, value structvar.View) error {
		return fn(path, TranslateModeLocalRelative, value)
	})
}
