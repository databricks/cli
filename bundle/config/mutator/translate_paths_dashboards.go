package mutator

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyDashboardTranslations(v dyn.Value) (dyn.Value, error) {
	// Convert the `file_path` field to a local absolute path.
	// We load the file at this path and use its contents for the dashboard contents.
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
		dyn.Key("file_path"),
	)

	return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for dashboard %s: %w", key, err)
		}

		return t.rewriteRelativeTo(p, v, t.retainLocalAbsoluteFilePath, dir, "")
	})
}
