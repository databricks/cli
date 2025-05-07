package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

func VisitDashboardPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
		dyn.Key("file_path"),
	)

	return dyn.MapByPattern(value, pattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		return fn(path, TranslateModeLocalRelative, value)
	})
}
