package terranova

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// represents node in the graph, each node is a resource
type nodeKey struct {
	Group string
	Name  string
	// Field names match deployplan.Action.
	// TODO: Here and in other places Name is ambiguous and should be replaced with ResourceKey
}

type fieldRef struct {
	field           dyn.Path // path to field within resource that contains the references, e.g. "description"
	ref             dynvar.Ref
	referencedNodes []nodeKey
}

func makeResourceGraph(ctx context.Context, b *bundle.Bundle, state resourcestate.ExportedResourcesMap, fieldRefsMap map[nodeKey][]fieldRef) (*dagrun.Graph[nodeKey], error) {
	g := dagrun.NewGraph[nodeKey]()

	// TODO: don't need a copy there
	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			group := p[1].Key()
			name := p[2].Key()

			_, ok := SupportedResources[group]
			if !ok {
				return v, fmt.Errorf("unsupported resource: %s", group)
			}

			groupState := state[group]
			delete(groupState, name)

			n := nodeKey{group, name}
			g.AddNode(n)

			fieldRefs, err := extractReferences(b.Config.Value(), n)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to read references from config: %w", err)
			}

			fieldRefsMap[n] = fieldRefs

			for _, fieldRef := range fieldRefs {
				for _, referencedNode := range fieldRef.referencedNodes {
					label := fmt.Sprintf("%s.%s -> %s.%s", referencedNode.Group, referencedNode.Name, n.Group, n.Name)
					log.Debugf(ctx, "Adding resource edge: %s (via %#v)", label, fieldRef.ref.Str)
					g.AddDirectedEdge(
						referencedNode,
						n,
						label,
					)
				}
			}
			return dyn.InvalidValue, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	return g, nil
}

func extractReferences(root dyn.Value, node nodeKey) ([]fieldRef, error) {
	var result []fieldRef

	val, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("resources"), dyn.Key(node.Group), dyn.Key(node.Name)))
	if err != nil {
		return nil, err
	}

	err = dyn.WalkReadOnly(val, func(p dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		referencedNodes, err := nodeFromRef(root, ref)
		if err != nil {
			return err
		}
		result = append(result, fieldRef{p, ref, referencedNodes})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing refs: %w", err)
	}
	return result, nil
}

func validateRef(root dyn.Value, ref string) (string, string, error) {
	items := strings.Split(ref, ".")
	if len(items) < 3 { // resources.jobs.foo.id
		return "", "", errors.New("reference too short")
	}
	if items[0] != "resources" {
		return "", "", errors.New("reference does not start with 'resources'")
	}
	_, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key(items[0]), dyn.Key(items[1]), dyn.Key(items[2])))
	if err != nil {
		return "", "", err
	}
	if len(items) > 4 || items[3] != "id" {
		return "", "", errors.New("${resources...} can only refer to field in the config or 'id'")
	}
	return items[1], items[2], nil
}

func nodeFromRef(root dyn.Value, ref dynvar.Ref) ([]nodeKey, error) {
	var referencedNodes []nodeKey
	for _, r := range ref.References() {
		// validateRef will check resource exists in the config; this will reject references to deleted resources, no need to handle that case separately.
		refGroup, refKey, err := validateRef(root, r)
		if err != nil {
			return nil, fmt.Errorf("cannot process reference %s: %w", r, err)
		}
		referencedNode := nodeKey{refGroup, refKey}
		referencedNodes = append(referencedNodes, referencedNode)
	}
	return referencedNodes, nil
}

func resolveIDReference(ctx context.Context, b *bundle.Bundle, group, resourceName string) error {
	mypath := dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key(group),
		dyn.Key(resourceName),
		dyn.Key("id"),
	)

	// log.Warnf(ctx, "resolving references node=%v IDIsReferenced=%v actionType=%v", node, IDIsReferenced, actionType)
	entry, hasEntry := b.ResourceDatabase.GetResourceEntry(group, resourceName)
	idValue := entry.ID
	if !hasEntry || idValue == "" {
		return errors.New("internal error: no db entry")
	}

	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
				if slices.Equal(path, mypath) {
					return dyn.V(idValue), nil
				}
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			})
		})
		if err != nil {
			logdiag.LogError(ctx, err)
		}
	})

	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {})

	if logdiag.HasError(ctx) {
		return errors.New("failed to update bundle config")
	}

	return nil
}
