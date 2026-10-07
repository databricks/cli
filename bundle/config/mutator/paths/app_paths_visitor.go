package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func VisitAppPaths(root structvar.View, fn VisitFunc) error {
	pattern := structpath.MustParsePattern("resources.apps.*.source_code_path")

	return structvar.ForEach(root, pattern, func(path *structpath.PathNode, value structvar.View) error {
		return fn(path, TranslateModeDirectory, value)
	})
}
