package paths

import (
	"github.com/databricks/cli/bundle/config"
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
	return nil
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
			if pl == nil || pl.Environment == nil {
				continue
			}
			for i := range pl.Environment.Dependencies {
				// Most dependencies are not local paths: check before building the path.
				if rule.skip(pl.Environment.Dependencies[i]) {
					continue
				}
				if err := visit(fn, visitArgs{
					value: &pl.Environment.Dependencies[i],
					mode:  rule.mode,
					path:  structpath.NewPath(resourcePath("pipelines", name), "environment", "dependencies", i),
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
