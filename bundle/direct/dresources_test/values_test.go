package dresources_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// absent is the "field not present in the config" value. Every transition is a move
// between two values drawn from a field's set, and add/remove are just the moves
// with absent on one side.
var absent = dyn.InvalidValue

// fieldValues is the per-resource-type value library, e.g. fields/schemas.yml.
//
// A field with no entry falls back to defaultValues for its Go kind. Fields the backend
// constrains (enums, ids, cross-referenced names) need an explicit entry, otherwise
// every value is rejected and the field reports only BACKEND_ERROR.
type fieldValues struct {
	// cloud enables this resource type in cloud runs. Off by default: a cloud run pays
	// real API latency per transition, so a type opts in once someone has checked that
	// it completes in reasonable time.
	cloud bool

	// skip lists field paths to leave out entirely, with a reason.
	skip map[string]string

	// fields maps a field path to the values to try.
	fields map[string][]dyn.Value
}

// loadFieldValues reads fields/<resource_type>.yml. Parsing goes through the repo's own
// yamlloader rather than a yaml package, so the values arrive as dyn.Value -- the same
// representation the bundle config uses, which is what they are written into.
func loadFieldValues(resourceType string) (*fieldValues, error) {
	fv := &fieldValues{cloud: false, skip: map[string]string{}, fields: map[string][]dyn.Value{}}

	path := filepath.Join("fields", resourceType+".yml")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fv, nil
	}
	if err != nil {
		return nil, err
	}

	root, err := yamlloader.LoadYAML(path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	if v, err := dyn.Get(root, "cloud"); err == nil {
		fv.cloud, _ = v.AsBool()
	}
	if v, err := dyn.Get(root, "skip"); err == nil {
		m, _ := v.AsMap()
		for _, pair := range m.Pairs() {
			key, _ := pair.Key.AsString()
			reason, _ := pair.Value.AsString()
			fv.skip[key] = reason
		}
	}
	if v, err := dyn.Get(root, "fields"); err == nil {
		m, _ := v.AsMap()
		for _, pair := range m.Pairs() {
			key, _ := pair.Key.AsString()
			values, ok := pair.Value.AsSequence()
			if !ok {
				return nil, fmt.Errorf("%s: fields.%s must be a list", path, key)
			}
			fv.fields[key] = values
		}
	}

	return fv, nil
}

// defaultValues per Go kind. Two values per kind is enough to observe a
// value->value transition on top of add and remove; more would multiply the matrix
// without testing a different code path.
//
// isContainer reports whether a kind has fields or elements to descend into.
func isContainer(kind reflect.Kind) bool {
	switch kind {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

//nolint:exhaustive // the default branch covers every kind without a generic value
func defaultValues(kind reflect.Kind) []dyn.Value {
	switch kind {
	case reflect.Bool:
		return []dyn.Value{dyn.V(false), dyn.V(true)}
	case reflect.String:
		return []dyn.Value{dyn.V("x"), dyn.V("y")}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return []dyn.Value{dyn.V(1), dyn.V(2)}
	case reflect.Float32, reflect.Float64:
		return []dyn.Value{dyn.V(1.0), dyn.V(2.0)}
	default:
		// The remaining kinds (interface, unsigned, complex, ...) have no meaningful
		// generic value; such a field needs an entry in the value library.
		return nil
	}
}

// field is one testable leaf of the resource's input struct.
type field struct {
	path   string // structpath/dyn path, e.g. "comment" or "email_notifications.on_failure"
	kind   reflect.Kind
	values []dyn.Value

	// required fields do not get an "absent" transition: a config missing one is
	// rejected by bundle validate, so removing it is not something a user can deploy.
	required bool
}

// transition is one move of a field from one value to another.
type transition struct {
	from, to dyn.Value
}

func (t transition) label() string {
	return valueLabel(t.from) + "->" + valueLabel(t.to)
}

// transitions returns every ordered pair of the field's values. absent is one of them,
// so adding and removing the field are the pairs with absent on one side.
func (f field) transitions() []transition {
	values := f.values
	if !f.required {
		values = append([]dyn.Value{absent}, values...)
	}
	var out []transition
	for _, from := range values {
		for _, to := range values {
			if valueLabel(from) != valueLabel(to) {
				out = append(out, transition{from: from, to: to})
			}
		}
	}
	return out
}

// requiredFields returns the field names the bundle schema marks required directly
// under the given path inside a resource, e.g. "" for the resource itself or
// "evaluation" for a nested object. The data is generated from the config structs
// (bundle/internal/validation), not from the engine's resources.yml.
func requiredFields(resourceType, parent string) []string {
	key := "resources." + resourceType + ".*"
	if parent != "" {
		key += "." + parent
	}
	return generated.RequiredFields[key]
}

func isRequired(resourceType, path string) bool {
	parent, name := "", path
	if i := strings.LastIndex(path, "."); i >= 0 {
		parent, name = path[:i], path[i+1:]
	}
	return slices.Contains(requiredFields(resourceType, parent), name)
}

// enumerateFields walks the resource's input config type the way cmd/bundle/debug
// refschema does, and pairs each scalar leaf with the values to try.
//
// Paths containing a wildcard are left out: they address a slice element or map entry
// that only exists if the base config happens to declare one, and picking an index
// here would silently test nothing. They are reported so the gap stays visible.
func enumerateFields(resourceType string, inputType reflect.Type, fv *fieldValues) (fields []field, wildcard []string) {
	_ = structwalk.WalkType(inputType, func(p *structpath.PatternNode, typ reflect.Type, sf *reflect.StructField) bool {
		if p.IsRoot() {
			return true
		}
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if isContainer(typ.Kind()) {
			// An intermediate node: descend to reach its leaves.
			return true
		}

		path := p.String()
		if strings.Contains(path, "*") {
			wildcard = append(wildcard, path)
			return false
		}
		if sf != nil && isNotUserSettable(sf) {
			return false
		}

		values := fv.fields[path]
		if values == nil {
			values = defaultValues(typ.Kind())
		}
		if values == nil {
			return false
		}
		fields = append(fields, field{
			path:     path,
			kind:     typ.Kind(),
			values:   values,
			required: isRequired(resourceType, path),
		})
		return false
	})
	return fields, wildcard
}

// isNotUserSettable reports whether the bundle marks a field as something the user
// never writes: an output the CLI fills in (id, url) or an internal bookkeeping field.
func isNotUserSettable(sf *reflect.StructField) bool {
	tag := sf.Tag.Get("bundle")
	return strings.Contains(tag, "readonly") ||
		strings.Contains(tag, "internal") ||
		sf.Tag.Get("json") == "-"
}
