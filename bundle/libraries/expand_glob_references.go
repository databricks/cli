package libraries

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type expand struct{}

func matchError(p *structpath.PathNode, l []diag.Location, message string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   message,
		Locations: l,
		Paths:     []*structpath.PathNode{p},
	}
}

func getLibDetails(lib compute.Library) (string, string, bool) {
	if lib.Whl != "" {
		return lib.Whl, "whl", true
	}

	if lib.Jar != "" {
		return lib.Jar, "jar", true
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

// expandFunc expands the item at path p into the items it is replaced with.
// If relocate is not empty, it names the field of each new item that takes the locations of the item.
type expandFunc[T any] func(ctx context.Context, b *bundle.Bundle, p *structpath.PathNode, item T) (output []T, relocate string, diags diag.Diagnostics)

// expandSequence replaces the items of the sequence at path p with the items expandFunc returns for them.
// New items keep the locations of the item they were expanded from.
func expandSequence[T any](ctx context.Context, b *bundle.Bundle, p *structpath.PathNode, lv structvar.View, fn expandFunc[T]) diag.Diagnostics {
	items, ok := sequencePointer[T](lv)
	if !ok {
		return nil
	}

	var diags diag.Diagnostics
	output := make([]T, 0, len(*items))
	var sources [][]int
	var relocates []string
	var locations [][]diag.Location
	for i, item := range *items {
		ip := structpath.NewIndex(p, i)
		expanded, relocate, d := fn(ctx, b, ip, item)
		diags = diags.Extend(d)
		locs := b.Config.LocationsAt(ip)
		for _, e := range expanded {
			output = append(output, e)
			sources = append(sources, []int{i})
			relocates = append(relocates, relocate)
			locations = append(locations, locs)
		}
	}

	*items = output
	b.Config.UpdateSequence(p, sources)
	for i, relocate := range relocates {
		if relocate != "" {
			b.Config.SetLocations(structpath.NewStringKey(structpath.NewIndex(p, i), relocate), locations[i])
		}
	}

	return diags
}

// sequencePointer returns a pointer to the typed sequence the view is based on.
func sequencePointer[T any](lv structvar.View) (*[]T, bool) {
	if lv.Kind() != structvar.KindSequence {
		return nil, false
	}

	v := lv.Reflect()
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if !v.CanAddr() {
		return nil, false
	}

	return reflect.TypeAssert[*[]T](v.Addr())
}

func expandLibraries(ctx context.Context, lv structvar.View, p *structpath.PathNode, b *bundle.Bundle) diag.Diagnostics {
	return expandSequence(ctx, b, p, lv, func(ctx context.Context, b *bundle.Bundle, ip *structpath.PathNode, lib compute.Library) ([]compute.Library, string, diag.Diagnostics) {
		path, libType, supported := getLibDetails(lib)
		if !supported || !IsLibraryLocal(path) {
			return []compute.Library{lib}, "", nil
		}

		matches, err := findMatches(ctx, b, path)
		if err != nil {
			return nil, "", diag.Diagnostics{matchError(structpath.NewStringKey(ip, libType), b.Config.LocationsAt(ip), err.Error())}
		}

		var output []compute.Library
		for _, match := range matches {
			if libType == "whl" {
				output = append(output, compute.Library{Whl: match})
			} else {
				output = append(output, compute.Library{Jar: match})
			}
		}
		return output, libType, nil
	})
}

func expandEnvironmentDeps(ctx context.Context, lv structvar.View, p *structpath.PathNode, b *bundle.Bundle) diag.Diagnostics {
	return expandSequence(ctx, b, p, lv, func(ctx context.Context, b *bundle.Bundle, ip *structpath.PathNode, path string) ([]string, string, diag.Diagnostics) {
		if !IsLibraryLocal(path) {
			return []string{path}, "", nil
		}

		// Strip extras before globbing so "[...]" isn't read as a glob class, then re-append.
		path, extras := patchwheel.SplitWheelExtras(path)

		matches, err := findMatches(ctx, b, path)
		if err != nil {
			return nil, "", diag.Diagnostics{matchError(ip, b.Config.LocationsAt(ip), err.Error())}
		}

		var output []string
		for _, match := range matches {
			output = append(output, match+extras)
		}
		return output, "", nil
	})
}

type expandPattern struct {
	pattern *structpath.PatternNode
	fn      func(ctx context.Context, lv structvar.View, p *structpath.PathNode, b *bundle.Bundle) diag.Diagnostics
}

var (
	taskLibrariesPattern              = structpath.MustParsePattern("resources.jobs.*.tasks[*].libraries")
	forEachTaskLibrariesPattern       = structpath.MustParsePattern("resources.jobs.*.tasks[*].for_each_task.task.libraries")
	aiRuntimeCodeSourcePattern        = structpath.MustParsePattern("resources.jobs.*.tasks[*].ai_runtime_task.code_source_path")
	forEachAiRuntimeCodeSourcePattern = structpath.MustParsePattern("resources.jobs.*.tasks[*].for_each_task.task.ai_runtime_task.code_source_path")
	envDepsPattern                    = structpath.MustParsePattern("resources.jobs.*.environments[*].spec.dependencies")
	pipelineEnvDepsPattern            = structpath.MustParsePattern("resources.pipelines.*.environment.dependencies")
	clusterLibrariesPattern           = structpath.MustParsePattern("resources.clusters.*.libraries")
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
		{
			pattern: clusterLibrariesPattern,
			fn:      expandLibraries,
		},
	}

	var diags diag.Diagnostics

	for _, expander := range expanders {
		err := structvar.ForEach(b.Config.View(), expander.pattern, func(p *structpath.PathNode, lv structvar.View) error {
			diags = diags.Extend(expander.fn(ctx, lv, p, b))
			return nil
		})
		if err != nil {
			diags = diags.Extend(diag.FromErr(err))
			break
		}
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
