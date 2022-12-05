package interpolation

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/databricks/bricks/bundle"
)

const Delimiter = "."

var re = regexp.MustCompile(`\$\{(\w+(\.\w+)*)\}`)

type stringField struct {
	path string

	getter
	setter
}

func newStringField(path string, g getter, s setter) *stringField {
	return &stringField{
		path: path,

		getter: g,
		setter: s,
	}
}

func (s *stringField) dependsOn() []string {
	var out []string
	m := re.FindAllStringSubmatch(s.Get(), -1)
	for i := range m {
		out = append(out, m[i][1])
	}
	return out
}

func (s *stringField) interpolate(fn LookupFunction, lookup map[string]string) {
	out := re.ReplaceAllStringFunc(s.Get(), func(s string) string {
		// Turn the whole match into the submatch.
		match := re.FindStringSubmatch(s)
		v, err := fn(match[1], lookup)
		if err != nil {
			panic(err)
		}

		return v
	})

	s.Set(out)
}

type accumulator struct {
	strings map[string]*stringField
}

// jsonFieldName returns the name in a field's `json` tag.
// Returns the empty string if it isn't set.
func jsonFieldName(sf reflect.StructField) string {
	tag, ok := sf.Tag.Lookup("json")
	if !ok {
		return ""
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return ""
	}
	return parts[0]
}

func (a *accumulator) walkStruct(scope []string, rv reflect.Value) {
	num := rv.NumField()
	for i := 0; i < num; i++ {
		sf := rv.Type().Field(i)
		f := rv.Field(i)

		// Walk field with the same scope for anonymous (embedded) fields.
		if sf.Anonymous {
			a.walk(scope, f, anySetter{f})
			continue
		}

		// Skip unnamed fields.
		fieldName := jsonFieldName(rv.Type().Field(i))
		if fieldName == "" {
			continue
		}

		a.walk(append(scope, fieldName), f, anySetter{f})
	}
}

func (a *accumulator) walk(scope []string, rv reflect.Value, s setter) {
	// Dereference pointer.
	if rv.Type().Kind() == reflect.Pointer {
		// Skip nil pointers.
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
		s = anySetter{rv}
	}

	switch rv.Type().Kind() {
	case reflect.String:
		path := strings.Join(scope, Delimiter)
		a.strings[path] = newStringField(path, anyGetter{rv}, s)
	case reflect.Struct:
		a.walkStruct(scope, rv)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			panic("only support string keys in map")
		}
		keys := rv.MapKeys()
		for _, key := range keys {
			a.walk(append(scope, key.String()), rv.MapIndex(key), mapSetter{rv, key})
		}
	case reflect.Slice:
		n := rv.Len()
		name := scope[len(scope)-1]
		base := scope[:len(scope)-1]
		for i := 0; i < n; i++ {
			element := rv.Index(i)
			a.walk(append(base, fmt.Sprintf("%s[%d]", name, i)), element, anySetter{element})
		}
	}
}

// Gathers the strings for a list of paths.
// The fields in these paths may not depend on other fields,
// as we don't support full DAG lookup yet (only single level).
func (a *accumulator) gather(paths []string) (map[string]string, error) {
	var out = make(map[string]string)
	for _, path := range paths {
		f, ok := a.strings[path]
		if !ok {
			return nil, fmt.Errorf("%s is not defined", path)
		}
		deps := f.dependsOn()
		if len(deps) > 0 {
			return nil, fmt.Errorf("%s depends on %s", path, strings.Join(deps, ", "))
		}
		out[path] = f.Get()
	}
	return out, nil
}

func (a *accumulator) start(v any) {
	rv := reflect.ValueOf(v)
	if rv.Type().Kind() != reflect.Pointer {
		panic("expect pointer")
	}
	rv = rv.Elem()
	if rv.Type().Kind() != reflect.Struct {
		panic("expect struct")
	}

	a.strings = make(map[string]*stringField)
	a.walk([]string{}, rv, nilSetter{})
}

func (a *accumulator) expand(fn LookupFunction) error {
	for path, v := range a.strings {
		ds := v.dependsOn()
		if len(ds) == 0 {
			continue
		}

		// Create map to be used for interpolation
		m, err := a.gather(ds)
		if err != nil {
			return fmt.Errorf("cannot interpolate %s: %w", path, err)
		}

		v.interpolate(fn, m)
	}

	return nil
}

type interpolate struct {
	fn LookupFunction
}

func (m *interpolate) expand(v any) error {
	a := accumulator{}
	a.start(v)
	return a.expand(m.fn)
}

func Interpolate(fn LookupFunction) bundle.Mutator {
	return &interpolate{fn: fn}
}

func (m *interpolate) Name() string {
	return "Interpolate"
}

func (m *interpolate) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	return nil, m.expand(&b.Config)
}
