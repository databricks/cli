package mutator

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyAppsTranslations(v dyn.Value) (dyn.Value, error) {
	// Convert the `source_code_path` field to a remote absolute path.
	// We use this path for app deployment to point to the source code.
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("source_code_path"),
	)

	return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for app %s: %w", key, err)
		}

		return t.rewriteRelativeTo(p, v, t.translateDirectoryPath, dir, "")
	})
}
