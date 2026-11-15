package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type capturePipelineDependencies struct{}

// CapturePipelineDependencies makes a pipeline wait for the grants and permissions of the resources
// it refers to, directly or through lifecycle.depends_on, so a pipeline is not created before the
// privileges it depends on are in place. A dependency that would create a cycle is skipped, e.g.
// when an app refers to the pipeline and its service principal is granted on the pipeline's schema.
func CapturePipelineDependencies() bundle.Mutator {
	return &capturePipelineDependencies{}
}

func (m *capturePipelineDependencies) Name() string {
	return "CapturePipelineDependencies"
}

func (m *capturePipelineDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	graph, err := dependencyGraph(b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Pipelines are visited in a fixed order, so the same cycle-forming dependency is skipped on every run.
	for _, key := range slices.Sorted(maps.Keys(b.Config.Resources.Pipelines)) {
		p := b.Config.Resources.Pipelines[key]
		node := "resources.pipelines." + key
		for _, dep := range graph[node] {
			for _, sub := range []string{".grants", ".permissions"} {
				target := dep + sub
				if _, ok := graph[target]; !ok || slices.Contains(graph[node], target) || reaches(graph, target, node) {
					continue
				}
				graph[node] = append(graph[node], target)
				p.Lifecycle.DependsOn = append(p.Lifecycle.DependsOn, resources.DependsOn{Resource: strings.TrimPrefix(target, "resources.")})
			}
		}
	}
	return nil
}

// dependencyGraph returns, for every resource and its grants or permissions, the nodes it depends
// on: those it refers to and those in its lifecycle.depends_on. Nodes are keyed as in the deployment
// plan, e.g. "resources.schemas.foo.grants"; a node without dependencies maps to nil.
func dependencyGraph(b *bundle.Bundle) (map[string][]string, error) {
	graph := make(map[string][]string)

	err := dyn.WalkReadOnly(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
		node, resourceType := config.GetNodeAndType(p)
		if resourceType == "" {
			return nil
		}

		src := node.String()
		if len(p) == len(node) {
			// Grants and permissions without entries are not deployed, so nothing can depend on them.
			if seq, ok := v.AsSequence(); len(node) == 4 && (!ok || len(seq) == 0) {
				return nil
			}
			if _, ok := graph[src]; !ok {
				graph[src] = nil
			}
			return nil
		}

		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			path, err := dyn.NewPathFromString(r)
			if err != nil {
				return fmt.Errorf("parsing %q: %w", r, err)
			}
			if target, targetType := config.GetNodeAndType(path); targetType != "" {
				graph[src] = append(graph[src], target.String())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, group := range b.Config.Resources.AllResources() {
		for key, resource := range group.Resources {
			// Python-generated configs can contain null resource entries.
			if reflect.ValueOf(resource).IsNil() {
				continue
			}
			src := "resources." + group.Description.PluralName + "." + key
			for _, dep := range resource.GetLifecycle().GetDependsOn() {
				// An entry that does not resolve is reported by validate.LifecycleDependsOn.
				if target, err := b.Config.Resources.ResolveDependsOn(dep.Resource); err == nil {
					graph[src] = append(graph[src], target)
				}
			}
		}
	}
	return graph, nil
}

// reaches reports whether from is to or depends on it, directly or through other nodes.
func reaches(graph map[string][]string, from, to string) bool {
	visited := make(map[string]bool)
	stack := []string{from}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == to {
			return true
		}
		if visited[n] {
			continue
		}
		visited[n] = true
		stack = append(stack, graph[n]...)
	}
	return false
}
