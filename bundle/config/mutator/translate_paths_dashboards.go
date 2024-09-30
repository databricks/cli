package mutator

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type dashboardRewritePattern struct {
	pattern dyn.Pattern
	fn      rewriteFunc
}

func (t *translateContext) dashboardRewritePatterns() []dashboardRewritePattern {
	// Base pattern to match all dashboards.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
	)

	// Compile list of configuration paths to rewrite.
	return []dashboardRewritePattern{
		{
			base.Append(dyn.Key("file_path")),
			t.retainLocalAbsoluteFilePath,
		},
	}
}

func (t *translateContext) applyDashboardTranslations(v dyn.Value) (dyn.Value, error) {
	var err error

	for _, rewritePattern := range t.dashboardRewritePatterns() {
		v, err = dyn.MapByPattern(v, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[1].Key()
			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for dashboard %s: %w", key, err)
			}

			return t.rewriteRelativeTo(p, v, rewritePattern.fn, dir, "")
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
