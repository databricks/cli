package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

func VisitAppPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("source_code_path"),
	)

	return dyn.MapByPattern(value, pattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		return fn(path, TranslateModeDirectory, value)
	})
}
