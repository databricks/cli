package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

func VisitAppPaths(root *config.Root, fn VisitFunc) error {
	names := sortedKeys(root.Resources.Apps)
	for _, name := range names {
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
	for _, name := range names {
		if a := root.Resources.Apps[name]; a != nil {
			if err := visit(fn, visitArgs{
				owner: a,
				field: "thumbnail_path",
				value: &a.ThumbnailPath,
				mode:  TranslateModeLocalRelative,
				path:  structpath.NewPath(resourcePath("apps", name), "thumbnail_path"),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
