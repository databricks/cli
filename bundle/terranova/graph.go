package terranova

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

type fieldRef struct {
	deployplan.ResourceNode
	Reference string // refrence in question e.g. ${resources.jobs.foo.id}
}

// makeResourceGraph creates node graph based on ${resources.group.name.id} references.
func makeResourceGraph(ctx context.Context, b *bundle.Bundle) (*dagrun.Graph[deployplan.ResourceNode], error) {
	g := dagrun.NewGraph[deployplan.ResourceNode]()

	// Collect and sort nodes first, because MapByPattern gives them in randomized order
	var nodes []deployplan.ResourceNode

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

			nodes = append(nodes, deployplan.ResourceNode{Group: group, Key: name})
			return dyn.InvalidValue, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	slices.SortFunc(nodes, func(a, b deployplan.ResourceNode) int {
		if a.Group == b.Group {
			return strings.Compare(a.Key, b.Key)
		}
		return strings.Compare(a.Group, b.Group)
	})

	for _, node := range nodes {
		g.AddNode(node)

		fieldRefs, err := extractReferences(b.Config.Value(), node)
		if err != nil {
			return nil, fmt.Errorf("failed to read references from config for %s: %w", node.String(), err)
		}

		for _, fieldRef := range fieldRefs {
			log.Debugf(ctx, "Adding resource edge: %s -> %s via %s", fieldRef.ResourceNode, node, fieldRef.Reference)
			// TODO: this may add duplicate edges. Investigate if we need to prevent that
			g.AddDirectedEdge(
				fieldRef.ResourceNode,
				node,
				fieldRef.Reference,
			)
		}
	}

	return g, nil
}

func extractReferences(root dyn.Value, node deployplan.ResourceNode) ([]fieldRef, error) {
	var result []fieldRef

	val, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("resources"), dyn.Key(node.Group), dyn.Key(node.Key)))
	if err != nil {
		return nil, err
	}

	err = dyn.WalkReadOnly(val, func(p dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			// validateRef will check resource exists in the config; this will reject references to deleted resources, no need to handle that case separately.
			edge, err := validateRef(root, r)
			if err != nil {
				return fmt.Errorf("cannot process reference %s: %w", r, err)
			}
			result = append(result, edge)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing refs: %w", err)
	}
	return result, nil
}

func validateRef(root dyn.Value, ref string) (fieldRef, error) {
	path, err := dyn.NewPathFromString(ref)
	if err != nil {
		return fieldRef{}, err
	}
	if len(path) < 3 { // expecting "resources.jobs.foo.*"
		return fieldRef{}, errors.New("reference too short")
	}
	if path[0].Key() != "resources" {
		return fieldRef{}, errors.New("reference does not start with 'resources'")
	}
	_, err = dyn.GetByPath(root, path[0:3])
	if err != nil {
		return fieldRef{}, err
	}
	return fieldRef{
		ResourceNode: deployplan.ResourceNode{
			Group: path[1].Key(),
			Key:   path[2].Key(),
		},
		Reference: "${" + ref + "}",
	}, nil
}

func resolveIDReference(ctx context.Context, b *bundle.Bundle, group, resourceName string) error {
	mypath := dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key(group),
		dyn.Key(resourceName),
		dyn.Key("id"),
	)

	entry, hasEntry := b.ResourceDatabase.GetResourceEntry(group, resourceName)
	idValue := entry.ID
	if !hasEntry || idValue == "" {
		return errors.New("internal error: no db entry")
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		root, err := dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
			if slices.Equal(path, mypath) {
				return dyn.V(idValue), nil
			}
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		})
		if err != nil {
			return root, err
		}
		// Following resolve_variable_references.go, normalize after variable substitution.
		// This fixes the following case: ${resources.jobs.foo.id} is replaced by string "12345"
		// This string corresponds to job_id integer field. Normalization converts "12345" to 12345.
		// Without normalization there will be an error when converting dynamic value to typed.
		root, diags := convert.Normalize(b.Config, root)
		for _, d := range diags {
			// TODO: add additional context if needed
			logdiag.LogDiag(ctx, d)
		}
		return root, nil
	})
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	if logdiag.HasError(ctx) {
		return errors.New("failed to update bundle config")
	}

	return nil
}

func resolveFieldReference(ctx context.Context, b *bundle.Bundle, targetPath dyn.Path, value any) error {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		root, err := dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
			if slices.Equal(path, targetPath) {
				return dyn.V(value), nil
			}
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		})
		if err != nil {
			return root, err
		}
		// Following resolve_variable_references.go, normalize after variable substitution.
		root, diags := convert.Normalize(b.Config, root)
		for _, d := range diags {
			logdiag.LogDiag(ctx, d)
		}
		return root, nil
	})
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	if logdiag.HasError(ctx) {
		return errors.New("failed to update bundle config")
	}

	return nil
}
