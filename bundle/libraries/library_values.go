package libraries

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

// visitLibraryValues calls fn for every library path value of the configuration, in
// this order: the wheels and jars of task libraries, of the libraries of tasks in
// for_each_task and of cluster libraries, the environment dependencies of jobs and of
// pipelines, and (codeSources) the code_source_path of AI Runtime tasks (also nested).
// Empty values are not visited. It stops at the first error fn returns.
func visitLibraryValues(r *config.Resources, codeSources bool, fn func(p *structpath.PathNode, s *string) error) error {
	v := &libraryValues{fn: fn}
	r.IterateTaskLibraries(false, v.wheels)
	r.IterateTaskLibraries(false, v.jars)
	r.IterateTaskLibraries(true, v.wheels)
	r.IterateTaskLibraries(true, v.jars)
	r.IterateClusterLibraries(v.wheels)
	r.IterateClusterLibraries(v.jars)
	r.IterateJobEnvironmentDependencies(v.dependencies)
	r.IteratePipelineEnvironmentDependencies(v.dependencies)
	if codeSources {
		r.IterateAiRuntimeCodeSources(false, v.value)
		r.IterateAiRuntimeCodeSources(true, v.value)
	}
	return v.err
}

// libraryValues visits library path values with fn until it returns an error.
type libraryValues struct {
	fn  func(p *structpath.PathNode, s *string) error
	err error
}

func (v *libraryValues) value(p *structpath.PathNode, s *string) bool {
	if v.err == nil && *s != "" {
		v.err = v.fn(p, s)
	}
	return v.err == nil
}

func (v *libraryValues) wheels(p *structpath.PathNode, libs *[]compute.Library) bool {
	for i := range *libs {
		if (*libs)[i].Whl != "" && !v.value(structpath.NewPath(p, i, "whl"), &(*libs)[i].Whl) {
			return false
		}
	}
	return v.err == nil
}

func (v *libraryValues) jars(p *structpath.PathNode, libs *[]compute.Library) bool {
	for i := range *libs {
		if (*libs)[i].Jar != "" && !v.value(structpath.NewPath(p, i, "jar"), &(*libs)[i].Jar) {
			return false
		}
	}
	return v.err == nil
}

func (v *libraryValues) dependencies(p *structpath.PathNode, deps *[]string) bool {
	for i := range *deps {
		if (*deps)[i] != "" && !v.value(structpath.NewIndex(p, i), &(*deps)[i]) {
			return false
		}
	}
	return v.err == nil
}
