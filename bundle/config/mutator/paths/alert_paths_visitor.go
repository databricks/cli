package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitAlertPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.Alerts) {
		if a := root.Resources.Alerts[name]; a != nil {
			if err := visit(a, "file_path", &a.FilePath, TranslateModeLocalRelative, nil, fn, func() *structpath.PathNode { return structpath.NewPath(resourcePath("alerts", name), "file_path") }); err != nil {
				return err
			}
		}
	}
	return nil
}
