package libraries

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/patchwheel"
)

type expand struct{}

func matchError(p dyn.Path, l []dyn.Location, message string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   message,
		Locations: l,
		Paths:     []dyn.Path{p},
	}
}

func getLibDetails(v dyn.Value) (string, string, bool) {
	m := v.MustMap()
	whl, ok := m.GetByString("whl")
	if ok {
		return whl.MustString(), "whl", true
	}

	jar, ok := m.GetByString("jar")
	if ok {
		return jar.MustString(), "jar", true
	}

	return "", "", false
}

func findMatches(ctx context.Context, b *bundle.Bundle, path string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(b.SyncRootPath, path))
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		if isGlobPattern(path) {
			return nil, fmt.Errorf("no files match pattern: %s", path)
		} else {
			return nil, fmt.Errorf("file doesn't exist %s", path)
		}
	}

	matches = patchwheel.FilterLatestWheels(ctx, matches)

	// We make the matched path relative to the sync root path before storing it
	// to allow upload mutator to distinguish between local and remote paths
	for i, match := range matches {
		matches[i], err = filepath.Rel(b.SyncRootPath, match)
		if err != nil {
			return nil, err
		}
	}

	return matches, nil
}

// Checks if the path is a glob pattern
// It can contain *, [] or ? characters
func isGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func expandLibraries(ctx context.Context, b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value) {
	var output []dyn.Value
	var diags diag.Diagnostics

	libs := v.MustSequence()
	for i, lib := range libs {
		lp := p.Append(dyn.Index(i))
		path, libType, supported := getLibDetails(lib)
		if !supported || !IsLibraryLocal(path) {
			output = append(output, lib)
			continue
		}

		lp = lp.Append(dyn.Key(libType))

		matches, err := findMatches(ctx, b, path)
		if err != nil {
			diags = diags.Append(matchError(lp, lib.Locations(), err.Error()))
			continue
		}

		for _, match := range matches {
			output = append(output, dyn.NewValue(map[string]dyn.Value{
				libType: dyn.NewValue(match, lib.Locations()),
			}, lib.Locations()))
		}
	}

	return diags, output
}

func expandEnvironmentDeps(ctx context.Context, b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value) {
	var output []dyn.Value
	var diags diag.Diagnostics

	deps := v.MustSequence()
	for i, dep := range deps {
		lp := p.Append(dyn.Index(i))
		path := dep.MustString()
		if !IsLibraryLocal(path) {
			output = append(output, dep)
			continue
		}

		matches, err := findMatches(ctx, b, path)
		if err != nil {
			diags = diags.Append(matchError(lp, dep.Locations(), err.Error()))
			continue
		}

		for _, match := range matches {
			output = append(output, dyn.NewValue(match, dep.Locations()))
		}
	}

	return diags, output
}

type expandPattern struct {
	pattern dyn.Pattern
	fn      func(ctx context.Context, b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value)
}

var taskLibrariesPattern = dyn.NewPattern(
	dyn.Key("resources"),
	dyn.Key("jobs"),
	dyn.AnyKey(),
	dyn.Key("tasks"),
	dyn.AnyIndex(),
	dyn.Key("libraries"),
)

var forEachTaskLibrariesPattern = dyn.NewPattern(
	dyn.Key("resources"),
	dyn.Key("jobs"),
	dyn.AnyKey(),
	dyn.Key("tasks"),
	dyn.AnyIndex(),
	dyn.Key("for_each_task"),
	dyn.Key("task"),
	dyn.Key("libraries"),
)

var envDepsPattern = dyn.NewPattern(
	dyn.Key("resources"),
	dyn.Key("jobs"),
	dyn.AnyKey(),
	dyn.Key("environments"),
	dyn.AnyIndex(),
	dyn.Key("spec"),
	dyn.Key("dependencies"),
)

var pipelineEnvDepsPattern = dyn.NewPattern(
	dyn.Key("resources"),
	dyn.Key("pipelines"),
	dyn.AnyKey(),
	dyn.Key("environment"),
	dyn.Key("dependencies"),
)

func (e *expand) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	expanders := []expandPattern{
		{
			pattern: taskLibrariesPattern,
			fn:      expandLibraries,
		},
		{
			pattern: forEachTaskLibrariesPattern,
			fn:      expandLibraries,
		},
		{
			pattern: envDepsPattern,
			fn:      expandEnvironmentDeps,
		},
		{
			pattern: pipelineEnvDepsPattern,
			fn:      expandEnvironmentDeps,
		},
	}

	var diags diag.Diagnostics

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, expander := range expanders {
			v, err = dyn.MapByPattern(v, expander.pattern, func(p dyn.Path, lv dyn.Value) (dyn.Value, error) {
				d, output := expander.fn(ctx, b, p, lv)
				diags = diags.Extend(d)
				return dyn.V(output), nil
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		return v, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}

func (e *expand) Name() string {
	return "libraries.ExpandGlobReferences"
}

// ExpandGlobReferences expands any glob references in the libraries or environments section
// to corresponding local paths.
// We only expand local paths (i.e. paths that are relative to the sync root path).
// After expanding we make the paths relative to the sync root path to allow upload mutator later in the chain to
// distinguish between local and remote paths.
func ExpandGlobReferences() bundle.Mutator {
	return &expand{}
}
