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
// privileges it depends on are in place. An entry that would make the pipeline wait on itself is
// skipped, e.g. when an app refers to the pipeline and its service principal is granted on the
// pipeline's schema.
func CapturePipelineDependencies() bundle.Mutator {
	return &capturePipelineDependencies{}
}

func (m *capturePipelineDependencies) Name() string {
	return "CapturePipelineDependencies"
}

func (m *capturePipelineDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Pipelines are visited in a fixed order, so the same entries are added on every run.
	for _, key := range slices.Sorted(maps.Keys(b.Config.Resources.Pipelines)) {
		p := b.Config.Resources.Pipelines[key]
		// Python-generated configs can contain null resource entries.
		if p == nil {
			continue
		}
		node := "resources.pipelines." + key
		deps, err := directDependencies(b, node)
		if err != nil {
			return diag.FromErr(err)
		}
		// A schema or volume is only usable with privileges on its catalog and schema too.
		candidates := slices.Clone(deps)
		for _, dep := range deps {
			parents, err := ucParents(b, dep)
			if err != nil {
				return diag.FromErr(err)
			}
			candidates = append(candidates, parents...)
		}
		for _, dep := range candidates {
			for _, sub := range []string{".grants", ".permissions"} {
				target := dep + sub
				if !hasPrivileges(b, target) || slices.Contains(deps, target) {
					continue
				}
				waits, err := reaches(b, target, node)
				if err != nil {
					return diag.FromErr(err)
				}
				if waits {
					continue
				}
				deps = append(deps, target)
				p.Lifecycle.DependsOn = append(p.Lifecycle.DependsOn, resources.DependsOn{Resource: strings.TrimPrefix(target, "resources.")})
			}
		}
	}
	return nil
}

// hasPrivileges reports whether the node is a grants or permissions list with entries. An empty list
// grants nothing and at most revokes earlier grants, so a pipeline gains nothing by waiting for it.
func hasPrivileges(b *bundle.Bundle, node string) bool {
	path, err := dyn.NewPathFromString(node)
	if err != nil {
		return false
	}
	v, err := dyn.GetByPath(b.Config.Value(), path)
	if err != nil {
		return false
	}
	seq, ok := v.AsSequence()
	return ok && len(seq) > 0
}

// directDependencies returns the nodes that the resource, or the grants or permissions, at node refers
// to, plus the nodes in its lifecycle.depends_on. Nodes are named as in the deployment plan, e.g.
// "resources.schemas.foo.grants". References from nested grants or permissions belong to those nodes,
// not to the resource.
func directDependencies(b *bundle.Bundle, node string) ([]string, error) {
	path, err := dyn.NewPathFromString(node)
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", node, err)
	}

	var deps []string
	if v, err := dyn.GetByPath(b.Config.Value(), path); err == nil {
		err := dyn.WalkReadOnly(v, func(rel dyn.Path, val dyn.Value) error {
			full := slices.Concat(path, rel)
			if src, typ := config.GetNodeAndType(full); typ == "" || src.String() != node {
				return nil
			}
			ref, ok := dynvar.NewRef(val)
			if !ok {
				return nil
			}
			for _, r := range ref.References() {
				refPath, err := dyn.NewPathFromString(r)
				if err != nil {
					return fmt.Errorf("parsing %q: %w", r, err)
				}
				if target, typ := config.GetNodeAndType(refPath); typ != "" {
					deps = append(deps, target.String())
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(path) == 3 {
		res, _, err := b.Config.Resources.ResolveKey(strings.TrimPrefix(node, "resources."))
		// Python-generated configs can contain null resource entries.
		if err == nil && !reflect.ValueOf(res).IsNil() {
			for _, dep := range res.GetLifecycle().GetDependsOn() {
				// An entry that does not resolve is reported by validate.LifecycleDependsOn.
				if target, err := b.Config.Resources.ResolveDependsOn(dep.Resource); err == nil {
					deps = append(deps, target)
				}
			}
		}
	}
	return deps, nil
}

// ucParents returns the catalogs and schemas that a schema or volume lives in, transitively, e.g. the
// schema and the catalog of a volume. Their grants are needed to use the object.
func ucParents(b *bundle.Bundle, node string) ([]string, error) {
	if !isUCObject(node) {
		return nil, nil
	}
	deps, err := directDependencies(b, node)
	if err != nil {
		return nil, err
	}
	var parents []string
	for _, dep := range deps {
		if !isUCContainer(dep) {
			continue
		}
		parents = append(parents, dep)
		more, err := ucParents(b, dep)
		if err != nil {
			return nil, err
		}
		parents = append(parents, more...)
	}
	return parents, nil
}

func isUCObject(node string) bool {
	return isTopLevelOf(node, "resources.schemas.") || isTopLevelOf(node, "resources.volumes.")
}

func isUCContainer(node string) bool {
	return isTopLevelOf(node, "resources.catalogs.") || isTopLevelOf(node, "resources.schemas.")
}

// isTopLevelOf reports whether node is a resource of the given type, e.g. "resources.schemas.foo"
// but not "resources.schemas.foo.grants".
func isTopLevelOf(node, prefix string) bool {
	rest, ok := strings.CutPrefix(node, prefix)
	return ok && rest != "" && !strings.Contains(rest, ".")
}

// reaches reports whether from is to or depends on it, directly or through other nodes. It is the
// guard against a cycle: a pipeline must not wait on a node that waits on the pipeline. It is only
// called for candidate entries, since directDependencies re-walks the whole resource on every visit.
func reaches(b *bundle.Bundle, from, to string) (bool, error) {
	visited := make(map[string]bool)
	stack := []string{from}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == to {
			return true, nil
		}
		if visited[n] {
			continue
		}
		visited[n] = true
		next, err := directDependencies(b, n)
		if err != nil {
			return false, err
		}
		stack = append(stack, next...)
	}
	return false, nil
}
