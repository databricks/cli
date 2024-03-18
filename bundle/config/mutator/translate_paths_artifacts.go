package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
)

func (m *translatePaths) applyArtifactTranslations(b *bundle.Bundle, v dyn.Value) (dyn.Value, error) {
	var err error

	// Base pattern to match all artifacts.
	base := dyn.NewPattern(
		dyn.Key("artifacts"),
		dyn.AnyKey(),
	)

	for _, t := range []struct {
		pattern dyn.Pattern
		fn      rewriteFunc
	}{
		{
			base.Append(dyn.Key("path")),
			translateNoOp,
		},
	} {
		v, err = dyn.MapByPattern(v, t.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[1].Key()
			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for artifact %s: %w", key, err)
			}

			return m.rewriteRelativeTo(b, p, v, t.fn, []string{dir})
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
