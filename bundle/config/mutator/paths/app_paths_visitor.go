package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitAppPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.Apps) {
		if a := root.Resources.Apps[name]; a != nil {
			if err := visit(a, "source_code_path", &a.SourceCodePath, TranslateModeDirectory, nil, fn, func() *structpath.PathNode { return structpath.NewPath(resourcePath("apps", name), "source_code_path") }); err != nil {
				return err
			}
		}
	}
	return nil
}
