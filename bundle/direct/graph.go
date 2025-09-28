package direct

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

type fieldRef struct {
	Node      string
	Reference string // refrence in question e.g. ${resources.jobs.foo.id}
}

// makeResourceGraph creates node graph based on ${resources.group.name.id} references.
func makeResourceGraph(ctx context.Context, configRoot dyn.Value) (*dagrun.Graph, error) {
	g := dagrun.NewGraph()

	// Collect and sort nodes first, because MapByPattern gives them in randomized order
	var nodes []string

	_, err := dyn.MapByPattern(
		configRoot,
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			group := p[1].Key()
			name := p[2].Key()

			_, ok := dresources.SupportedResources[group]
			if !ok {
				return v, fmt.Errorf("unsupported resource: %s", group)
			}

			nodes = append(nodes, "resources."+group+"."+name)
			return dyn.InvalidValue, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	slices.Sort(nodes)

	for _, node := range nodes {
		g.AddNode(node)

		fieldRefs, err := extractReferences(configRoot, node)
		if err != nil {
			return nil, fmt.Errorf("failed to read references from config for %s: %w", node, err)
		}

		for _, fieldRef := range fieldRefs {
			log.Debugf(ctx, "Adding resource edge: %s -> %s via %s", fieldRef.Node, node, fieldRef.Reference)
			// TODO: this may add duplicate edges. Investigate if we need to prevent that
			g.AddDirectedEdge(
				fieldRef.Node,
				node,
				fieldRef.Reference,
			)
		}
	}

	return g, nil
}

func extractReferences(root dyn.Value, node string) ([]fieldRef, error) {
	var result []fieldRef

	path, err := dyn.NewPathFromString(node)
	if err != nil {
		return nil, fmt.Errorf("internal error: bad node key: %q: %w", node, err)
	}

	val, err := dyn.GetByPath(root, path)
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
			item, err := validateRef(root, r)
			if err != nil {
				return fmt.Errorf("cannot process reference %s: %w", r, err)
			}
			result = append(result, item)
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
		Node:      "resources." + path[1].Key() + "." + path[2].Key(),
		Reference: "${" + ref + "}",
	}, nil
}

// adaptValue converts arbitrary values to types that dyn library can handle.
// The dyn library supports basic Go types (string, bool, int, float) but not typedefs.
// This function normalizes SDK typedefs to their underlying representation.
func adaptValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.String:
		return rv.String(), nil
	case reflect.Bool:
		return rv.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil
	case reflect.Ptr:
		if rv.IsNil() {
			return nil, nil
		}
		return adaptValue(rv.Elem().Interface())
	case reflect.Interface:
		if rv.IsNil() {
			return nil, nil
		}
		return adaptValue(rv.Elem().Interface())
	default:
		return nil, fmt.Errorf("unsupported type %T (kind %v)", value, rv.Kind())
	}
}

func replaceReferenceWithValue(ctx context.Context, bundleConfig *config.Root, reference string, value any) error {
	targetPath, ok := dynvar.PureReferenceToPath(reference)
	if !ok {
		return fmt.Errorf("internal error: bad reference: %q", reference)
	}

	// dyn modules does not work with typedefs, only original types; SDK have many typedefs, so we simplify type here
	// adaptValue should also return for non-scalar types like structs and maps and slices
	normValue, err := adaptValue(value)
	if err != nil {
		return fmt.Errorf("cannot resolve value of type %T: %w", value, err)
	}

	err = bundleConfig.Mutate(func(root dyn.Value) (dyn.Value, error) {
		root, err := dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
			if slices.Equal(path, targetPath) {
				return dyn.V(normValue), nil
			}
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		})
		if err != nil {
			return root, err
		}
		// Following resolve_variable_references.go, normalize after variable substitution.
		root, diags := convert.Normalize(bundleConfig, root)
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
