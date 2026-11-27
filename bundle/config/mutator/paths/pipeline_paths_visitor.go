package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// pipelineLibraryRule visits one kind of path value of the libraries of a pipeline.
type pipelineLibraryRule struct {
	mode  TranslateMode
	keys  []any
	field func(*pipelines.PipelineLibrary) (owner any, s *string)
}

var pipelineLibraryRules = []pipelineLibraryRule{
	{TranslateModeNotebook, []any{"notebook", "path"}, func(l *pipelines.PipelineLibrary) (any, *string) {
		if l.Notebook == nil {
			return nil, nil
		}
		return l.Notebook, &l.Notebook.Path
	}},
	{TranslateModeFile, []any{"file", "path"}, func(l *pipelines.PipelineLibrary) (any, *string) {
		if l.File == nil {
			return nil, nil
		}
		return l.File, &l.File.Path
	}},
	{TranslateModeGlob, []any{"glob", "include"}, func(l *pipelines.PipelineLibrary) (any, *string) {
		if l.Glob == nil {
			return nil, nil
		}
		return l.Glob, &l.Glob.Include
	}},
}

func VisitPipelinePaths(root *config.Root, fn VisitFunc) error {
	for _, rule := range pipelineLibraryRules {
		for _, name := range sortedKeys(root.Resources.Pipelines) {
			pl := root.Resources.Pipelines[name]
			if pl == nil {
				continue
			}
			for i := range pl.Libraries {
				owner, s := rule.field(&pl.Libraries[i])
				if s == nil {
					continue
				}
				if err := visit(fn, visitArgs{
					owner: owner,
					field: rule.keys[len(rule.keys)-1].(string),
					value: s,
					mode:  rule.mode,
					path:  structpath.NewPath(structpath.NewPath(resourcePath("pipelines", name), "libraries", i), rule.keys...),
				}); err != nil {
					return err
				}
			}
		}
	}
	for _, name := range sortedKeys(root.Resources.Pipelines) {
		if pl := root.Resources.Pipelines[name]; pl != nil {
			if err := visit(fn, visitArgs{
				owner: pl,
				field: "root_path",
				value: &pl.RootPath,
				mode:  TranslateModeDirectory,
				path:  structpath.NewPath(resourcePath("pipelines", name), "root_path"),
			}); err != nil {
				return err
			}
		}
	}
	for _, name := range sortedKeys(root.Resources.Pipelines) {
		pl := root.Resources.Pipelines[name]
		if pl == nil || pl.Testing == nil {
			continue
		}
		for i := range pl.Testing.Sources {
			g := pl.Testing.Sources[i].Glob
			if g == nil {
				continue
			}
			if err := visit(fn, visitArgs{
				owner: g,
				field: "include",
				value: &g.Include,
				mode:  TranslateModeGlob,
				path:  structpath.NewPath(resourcePath("pipelines", name), "testing", "sources", i, "glob", "include"),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// dependencyList is a list of environment dependencies and the keys of its path in the pipeline.
type dependencyList struct {
	keys []any
	deps []string
}

// pipelineDependencies returns the environment dependencies of a pipeline. Test dependencies
// take the same forms as the pipeline's environment dependencies.
func pipelineDependencies(pl *resources.Pipeline) []dependencyList {
	var out []dependencyList
	if pl.Environment != nil {
		out = append(out, dependencyList{[]any{"environment", "dependencies"}, pl.Environment.Dependencies})
	}
	if pl.Testing != nil && pl.Testing.Environment != nil {
		out = append(out, dependencyList{[]any{"testing", "environment", "dependencies"}, pl.Testing.Environment.Dependencies})
	}
	return out
}

func VisitPipelineLibrariesPaths(root *config.Root, fn VisitFunc) error {
	for _, rule := range []struct {
		mode TranslateMode
		skip func(string) bool
	}{
		{TranslateModeLocalRelativeWithPrefix, notLocalLibrary},
		{TranslateModeEnvironmentPipFlag, noLocalPathInPipFlag},
	} {
		for _, name := range sortedKeys(root.Resources.Pipelines) {
			pl := root.Resources.Pipelines[name]
			if pl == nil {
				continue
			}
			for _, d := range pipelineDependencies(pl) {
				for i := range d.deps {
					// Most dependencies are not local paths: check before building the path.
					if rule.skip(d.deps[i]) {
						continue
					}
					if err := visit(fn, visitArgs{
						value: &d.deps[i],
						mode:  rule.mode,
						path:  structpath.NewPath(structpath.NewPath(resourcePath("pipelines", name), d.keys...), i),
					}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
