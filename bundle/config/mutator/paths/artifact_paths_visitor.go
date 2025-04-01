package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

type artifactRewritePattern struct {
	pattern dyn.Pattern
	mode    TranslateMode
}

func artifactRewritePatterns() []artifactRewritePattern {
	// Base pattern to match all artifacts.
	base := dyn.NewPattern(
		dyn.Key("artifacts"),
		dyn.AnyKey(),
	)

	// Compile list of configuration paths to rewrite.
	return []artifactRewritePattern{
		{
			pattern: base.Append(dyn.Key("path")),
			mode:    TranslateModeLocalAbsoluteDirectory,
		},
	}
}

func VisitArtifactPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	var err error
	newValue := value

	for _, rewritePattern := range artifactRewritePatterns() {
		newValue, err = dyn.MapByPattern(newValue, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return fn(p, rewritePattern.mode, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newValue, nil
}
