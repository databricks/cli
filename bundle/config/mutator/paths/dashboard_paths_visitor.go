package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitDashboardPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.Dashboards) {
		if d := root.Resources.Dashboards[name]; d != nil {
			if err := visit(d, "file_path", &d.FilePath, TranslateModeLocalRelative, nil, fn, func() *structpath.PathNode { return structpath.NewPath(resourcePath("dashboards", name), "file_path") }); err != nil {
				return err
			}
		}
	}
	return nil
}
