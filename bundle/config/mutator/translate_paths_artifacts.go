package mutator

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type artifactRewritePattern struct {
	pattern dyn.Pattern
	fn      rewriteFunc
}

func (t *translateContext) artifactRewritePatterns() []artifactRewritePattern {
	// Base pattern to match all artifacts.
	base := dyn.NewPattern(
		dyn.Key("artifacts"),
		dyn.AnyKey(),
	)

	// Compile list of configuration paths to rewrite.
	return []artifactRewritePattern{
		{
			base.Append(dyn.Key("path")),
			t.translateNoOp,
		},
	}
}

func (t *translateContext) applyArtifactTranslations(v dyn.Value) (dyn.Value, error) {
	var err error

	for _, rewritePattern := range t.artifactRewritePatterns() {
		v, err = dyn.MapByPattern(v, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[1].Key()
			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for artifact %s: %w", key, err)
			}

			return t.rewriteRelativeTo(p, v, rewritePattern.fn, dir, "")
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
