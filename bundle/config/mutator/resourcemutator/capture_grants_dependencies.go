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
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type captureGrantsDependencies struct{}

// CaptureGrantsDependencies makes a resource that depends on another resource also depend on
// that resource's grants, because creating some resources requires the grants to be in place:
// a pipeline cannot be created before its run_as principal has CREATE TABLE on the target
// schema. The dependency is added to lifecycle.depends_on, and skipped when it would create a
// cycle, e.g. when a schema grants privileges to the service principal of an app that refers
// to the schema.
func CaptureGrantsDependencies() bundle.Mutator {
	return &captureGrantsDependencies{}
}

func (m *captureGrantsDependencies) Name() string {
	return "CaptureGrantsDependencies"
}

func (m *captureGrantsDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	graph, err := dependencyGraph(b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Resources are visited in a fixed order because an added dependency can rule out a later
	// one: with two of them forming a cycle together, the same one must be skipped on every run.
	added := make(map[string][]string)
	for _, node := range slices.Sorted(maps.Keys(graph)) {
		// Grants and permissions have no lifecycle.
		if strings.Contains(config.GetResourceTypeFromKey(node), ".") {
			continue
		}

		// The grants appended below are not visited: range evaluates the slice once.
		for _, dep := range graph[node] {
			grants := dep + ".grants"
			if _, ok := graph[grants]; !ok || slices.Contains(graph[node], grants) || reaches(graph, grants, node) {
				continue
			}
			graph[node] = append(graph[node], grants)
			added[node] = append(added[node], grants)
		}
	}

	resources := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, resources, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			grants := added[p.String()]
			if len(grants) == 0 {
				return v, nil
			}
			return appendDependsOn(v, grants)
		})
	})
	return diag.FromErr(err)
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

// appendDependsOn appends lifecycle.depends_on entries for the given nodes to a resource.
func appendDependsOn(resource dyn.Value, nodes []string) (dyn.Value, error) {
	lifecycle := dyn.GetValue(resource, "lifecycle")
	if lifecycle.Kind() != dyn.KindMap {
		lifecycle = dyn.V(dyn.NewMapping())
	}

	entries, _ := dyn.GetValue(lifecycle, "depends_on").AsSequence()
	for _, node := range nodes {
		entries = append(entries, dyn.V(map[string]dyn.Value{
			"resource": dyn.V(strings.TrimPrefix(node, "resources.")),
		}))
	}

	lifecycle, err := dyn.Set(lifecycle, "depends_on", dyn.V(entries))
	if err != nil {
		return dyn.InvalidValue, err
	}
	return dyn.Set(resource, "lifecycle", lifecycle)
}
