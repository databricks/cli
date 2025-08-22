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
	field           dyn.Path // path to field within resource that contains the references, e.g. "description"
	ref             dynvar.Ref
	referencedNodes []deployplan.ResourceNode
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
			for _, referencedNode := range fieldRef.referencedNodes {
				// We're only supporting "id" field at the moment, so label is unambigous
				label := "${resources." + referencedNode.Group + "." + referencedNode.Key + ".id}"
				log.Debugf(ctx, "Adding resource edge: %s (via %#v)", label, fieldRef.ref.Str)
				// TODO: this may add duplicate edges. Investigate if we need to prevent that
				g.AddDirectedEdge(
					referencedNode,
					node,
					label,
				)
			}
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
		return "", "", errors.New("only ${resources.<group>.<key>.id} references are supported")
	}
	return items[1], items[2], nil
}

func nodeFromRef(root dyn.Value, ref dynvar.Ref) ([]deployplan.ResourceNode, error) {
	var referencedNodes []deployplan.ResourceNode
	for _, r := range ref.References() {
		// validateRef will check resource exists in the config; this will reject references to deleted resources, no need to handle that case separately.
		refGroup, refKey, err := validateRef(root, r)
		if err != nil {
			return nil, fmt.Errorf("cannot process reference %s: %w", r, err)
		}
		referencedNode := deployplan.ResourceNode{
			Group: refGroup,
			Key:   refKey,
		}
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
