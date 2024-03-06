package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
)

func (m *translatePaths) applyPipelineTranslations(b *bundle.Bundle, v dyn.Value) (dyn.Value, error) {
	var fallback = make(map[string]string)
	var err error

	for key, pipeline := range b.Config.Resources.Pipelines {
		dir, err := pipeline.ConfigFileDirectory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
		}

		// If we cannot resolve the relative path using the [dyn.Value] location itself,
		// use the pipeline's location as fallback. This is necessary for backwards compatibility.
		fallback[key] = dir
	}

	// Base pattern to match all libraries in all pipelines.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("pipelines"),
		dyn.AnyKey(),
		dyn.Key("libraries"),
		dyn.AnyIndex(),
	)

	for _, t := range []struct {
		pattern dyn.Pattern
		fn      rewriteFunc
	}{
		{
			base.Append(dyn.Key("notebook"), dyn.Key("path")),
			translateNotebookPath,
		},
		{
			base.Append(dyn.Key("file"), dyn.Key("path")),
			translateFilePath,
		},
	} {
		v, err = dyn.MapByPattern(v, t.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return m.rewriteValue(b, p, v, t.fn)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
