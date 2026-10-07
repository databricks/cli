package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type artifactRewritePattern struct {
	pattern *structpath.PatternNode
	mode    TranslateMode
}

func artifactRewritePatterns() []artifactRewritePattern {
	// Base pattern to match all artifacts.
	base := "artifacts.*"

	// Compile list of configuration paths to rewrite.
	return []artifactRewritePattern{
		{
			pattern: structpath.MustParsePattern(base + ".path"),
			mode:    TranslateModeLocalAbsoluteDirectory,
		},
	}
}

func VisitArtifactPaths(root structvar.View, fn VisitFunc) error {
	for _, rewritePattern := range artifactRewritePatterns() {
		err := structvar.ForEach(root, rewritePattern.pattern, func(p *structpath.PathNode, v structvar.View) error {
			return fn(p, rewritePattern.mode, v)
		})
		if err != nil {
			return err
		}
	}

	return nil
}
