package libraries

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type expand struct {
}

func matchWarning(p dyn.Path, message string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  message,
		Paths: []dyn.Path{
			p.Append(),
		},
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

func findMatches(b *bundle.Bundle, path string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(b.RootPath, path))
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matching files for %s", path)
	}

	return matches, nil
}

func expandLibraries(b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value) {
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

		matches, err := findMatches(b, path)
		if err != nil {
			diags = diags.Append(matchWarning(lp, err.Error()))
			continue
		}

		for _, match := range matches {
			output = append(output, dyn.NewValue(map[string]dyn.Value{
				libType: dyn.V(match),
			}, lib.Locations()))
		}
	}

	return diags, output
}

func expandEnvironmentDeps(b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value) {
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

		matches, err := findMatches(b, path)
		if err != nil {
			diags = diags.Append(matchWarning(lp, err.Error()))
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
	fn      func(b *bundle.Bundle, p dyn.Path, v dyn.Value) (diag.Diagnostics, []dyn.Value)
}

func (e *expand) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	taskLibraries := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
		dyn.Key("tasks"),
		dyn.AnyIndex(),
		dyn.Key("libraries"),
	)

	forEachTaskLibraries := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
		dyn.Key("tasks"),
		dyn.AnyIndex(),
		dyn.Key("for_each_task"),
		dyn.Key("task"),
		dyn.Key("libraries"),
	)

	envDeps := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
		dyn.Key("environments"),
		dyn.AnyIndex(),
		dyn.Key("spec"),
		dyn.Key("dependencies"),
	)

	expanders := []expandPattern{
		{
			pattern: taskLibraries,
			fn:      expandLibraries,
		},
		{
			pattern: forEachTaskLibraries,
			fn:      expandLibraries,
		},
		{
			pattern: envDeps,
			fn:      expandEnvironmentDeps,
		},
	}

	var diags diag.Diagnostics

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, expander := range expanders {
			v, err = dyn.MapByPattern(v, expander.pattern, func(p dyn.Path, lv dyn.Value) (dyn.Value, error) {
				d, output := expander.fn(b, p, lv)
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
// to corresponding local paths
func ExpandGlobReferences() bundle.Mutator {
	return &expand{}
}
