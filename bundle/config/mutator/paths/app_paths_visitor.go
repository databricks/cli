package paths

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
)

func VisitAppPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	sourceCodePattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("source_code_path"),
	)

	value, err := dyn.MapByPattern(value, sourceCodePattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		return fn(path, resources.TranslateModeDirectory, value)
	})
	if err != nil {
		return dyn.InvalidValue, err
	}

	thumbnailPattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("thumbnail_path"),
	)

	return dyn.MapByPattern(value, thumbnailPattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		return fn(path, resources.TranslateModeLocalRelative, value)
	})
}
