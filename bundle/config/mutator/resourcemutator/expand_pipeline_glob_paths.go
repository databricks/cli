package resourcemutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/patchwheel"
)

type expandPipelineGlobPaths struct{}

func ExpandPipelineGlobPaths() bundle.Mutator {
	return &expandPipelineGlobPaths{}
}

func (m *expandPipelineGlobPaths) expandLibrary(ctx context.Context, dir string, v dyn.Value) ([]dyn.Value, error) {
	// Probe for the path field in the library.
	for _, p := range []dyn.Path{
		dyn.NewPath(dyn.Key("notebook"), dyn.Key("path")),
		dyn.NewPath(dyn.Key("file"), dyn.Key("path")),
	} {
		pv, err := dyn.GetByPath(v, p)
		if dyn.IsNoSuchKeyError(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		// If the path is empty or not a local path, return the original value.
		path := pv.MustString()
		if path == "" || !libraries.IsLocalPath(path) {
			return []dyn.Value{v}, nil
		}

		matches, err := filepath.Glob(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}

		// If there are no matches, return the original value.
		if len(matches) == 0 {
			return []dyn.Value{v}, nil
		}

		matches = patchwheel.FilterLatestWheels(ctx, matches)

		// Emit a new value for each match.
		var ev []dyn.Value
		for _, match := range matches {
			m, err := filepath.Rel(dir, match)
			if err != nil {
				return nil, err
			}
			nv, err := dyn.SetByPath(v, p, dyn.NewValue(filepath.ToSlash(m), pv.Locations()))
			if err != nil {
				return nil, err
			}
			ev = append(ev, nv)
		}

		return ev, nil
	}

	// Neither of the library paths were found. This is likely an invalid node,
	// but it isn't this mutator's job to enforce that. Return the original value.
	return []dyn.Value{v}, nil
}

func (m *expandPipelineGlobPaths) expandSequence(ctx context.Context, dir string, p dyn.Path, v dyn.Value) (dyn.Value, error) {
	s, ok := v.AsSequence()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected sequence, got %s", v.Kind())
	}

	var vs []dyn.Value
	for _, sv := range s {
		v, err := m.expandLibrary(ctx, dir, sv)
		if err != nil {
			return dyn.InvalidValue, err
		}

		vs = append(vs, v...)
	}

	return dyn.NewValue(vs, v.Locations()), nil
}

func (m *expandPipelineGlobPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		p := dyn.NewPattern(
			dyn.Key("resources"),
			dyn.Key("pipelines"),
			dyn.AnyKey(),
			dyn.Key("libraries"),
		)

		// Visit each pipeline's "libraries" field and expand any glob patterns.
		return dyn.MapByPattern(v, p, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return m.expandSequence(ctx, b.BundleRootPath, path, value)
		})
	})

	return diag.FromErr(err)
}

func (*expandPipelineGlobPaths) Name() string {
	return "ExpandPipelineGlobPaths"
}
