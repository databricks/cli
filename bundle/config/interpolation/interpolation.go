package interpolation

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/databricks/bricks/bundle"
)

var re = regexp.MustCompile(`\$\{(\w+(\.\w+)*)\}`)

type stringField struct {
	rv reflect.Value
	s  setter
}

func newStringField(path string, rv reflect.Value, s setter) *stringField {
	return &stringField{
		rv: rv,
		s:  s,
	}
}

func (s *stringField) String() string {
	return s.rv.String()
}

func (s *stringField) dependsOn() []string {
	var out []string
	m := re.FindAllStringSubmatch(s.String(), -1)
	for i := range m {
		out = append(out, m[i][1])
	}
	return out
}

func (s *stringField) interpolate(lookup map[string]string) {
	out := re.ReplaceAllStringFunc(s.String(), func(s string) string {
		// Turn the whole match into the submatch.
		match := re.FindStringSubmatch(s)
		path := match[1]
		v, ok := lookup[path]
		if !ok {
			panic(fmt.Sprintf("expected to find value for path: %s", path))
		}
		return v
	})

	s.s.Set(out)
}

type accumulator struct {
	strings map[string]*stringField
}

func jsonFieldName(sf reflect.StructField) *string {
	tag, ok := sf.Tag.Lookup("json")
	if !ok {
		return nil
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return nil
	}
	return &parts[0]
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

		// Skip fields without a JSON tag.
		name := jsonFieldName(rv.Type().Field(i))
		if name == nil {
			continue
		}

		a.walk(append(scope, *name), f, anySetter{f})
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
		path := strings.Join(scope, ".")
		a.strings[path] = newStringField(path, rv, s)
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
		out[path] = f.rv.String()
	}
	return out, nil
}

func expand(v any) {
	rv := reflect.ValueOf(v)
	if rv.Type().Kind() != reflect.Pointer {
		panic("expect pointer")
	}
	rv = rv.Elem()
	if rv.Type().Kind() != reflect.Struct {
		panic("expect struct")
	}
	acc := &accumulator{
		strings: make(map[string]*stringField),
	}
	acc.walk([]string{}, rv, nilSetter{})

	for path, v := range acc.strings {
		ds := v.dependsOn()
		if len(ds) == 0 {
			continue
		}

		// Create map to be used for interpolation
		m, err := acc.gather(ds)
		if err != nil {
			panic(fmt.Errorf("cannot interpolate %s: %w", path, err))
		}

		v.interpolate(m)
	}
}

type interpolate struct{}

func Interpolate() bundle.Mutator {
	return &interpolate{}
}

func (m *interpolate) Name() string {
	return "Interpolate"
}

func (m *interpolate) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	expand(&b.Config)
	return nil, nil
}
