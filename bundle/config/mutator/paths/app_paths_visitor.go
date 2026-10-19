package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitAppPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.Apps) {
		if a := root.Resources.Apps[name]; a != nil {
			if err := visit(fn, visitArgs{
				owner: a,
				field: "source_code_path",
				value: &a.SourceCodePath,
				mode:  TranslateModeDirectory,
				path:  structpath.NewPath(resourcePath("apps", name), "source_code_path"),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
