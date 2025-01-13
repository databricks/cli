package mutator

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyJobTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	var err error

	fallback, err := gatherFallbackPaths(v, "jobs")
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Do not translate job task paths if using Git source
	var ignore []string
	for key, job := range t.b.Config.Resources.Jobs {
		if job.GitSource != nil {
			ignore = append(ignore, key)
		}
	}

	return paths.VisitJobPaths(v, func(p dyn.Path, kind paths.PathKind, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()

		// Skip path translation if the job is using git source.
		if slices.Contains(ignore, key) {
			return v, nil
		}

		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
		}

		rewritePatternFn, err := t.getRewritePatternFn(kind)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return t.rewriteRelativeTo(ctx, p, v, rewritePatternFn, dir, fallback[key])
	})
}

func (t *translateContext) getRewritePatternFn(kind paths.PathKind) (rewriteFunc, error) {
	switch kind {
	case paths.PathKindLibrary:
		return t.translateNoOp, nil
	case paths.PathKindNotebook:
		return t.translateNotebookPath, nil
	case paths.PathKindWorkspaceFile:
		return t.translateFilePath, nil
	case paths.PathKindDirectory:
		return t.translateDirectoryPath, nil
	case paths.PathKindWithPrefix:
		return t.translateNoOpWithPrefix, nil
	}

	return nil, fmt.Errorf("unsupported path kind: %d", kind)
}
