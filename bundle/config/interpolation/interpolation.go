package interpolation

import (
	"context"
	"errors"
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

func (s *stringField) interpolate(fns []LookupFunction, lookup map[string]string) {
	out := re.ReplaceAllStringFunc(s.Get(), func(s string) string {
		// Turn the whole match into the submatch.
		match := re.FindStringSubmatch(s)
		for _, fn := range fns {
			v, err := fn(match[1], lookup)
			if errors.Is(err, ErrSkipInterpolation) {
				continue
			}
			if err != nil {
				panic(err)
			}
			return v
		}

		// No substitution.
		return s
	})

	s.Set(out)
}

type accumulator struct {
	// all string fields in the bundle config
	strings map[string]*stringField

	// contains path -> resolve string mapping for string fields in the config
	memo map[string]string
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

// walk and gather all string fields in the config
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
	a.memo = make(map[string]string)
	a.walk([]string{}, rv, nilSetter{})
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (a *accumulator) SafeResolve(path string, seenPaths *[]string, fns ...LookupFunction) error {
	// error if there is a loop in variable interpolation
	if contains(*seenPaths, path) {
		return fmt.Errorf("cycle retected in field resolution: %s -> %s", strings.Join(*seenPaths, " -> "), path)
	}

	// add path to seenPaths
	*seenPaths = append(*seenPaths, path)

	// resolve the string field at path
	err := a.Resolve(path, *seenPaths, fns...)
	if err != nil {
		return err
	}

	// remove path from seenPaths
	*seenPaths = (*seenPaths)[:len(*seenPaths)-1]
	return nil
}

// recursively interpolate variables in a depth first manner
func (a *accumulator) Resolve(path string, seenPaths []string, fns ...LookupFunction) error {
	// return early if the path is already resolved
	if _, ok := a.memo[path]; ok {
		return nil
	}

	// fetch the string node to resolve
	rootField, ok := a.strings[path]
	if !ok {
		return fmt.Errorf("could not find string field with path %s", path)
	}

	// return early if the string field has no variables to interpolate
	if len(rootField.dependsOn()) == 0 {
		a.memo[path] = rootField.Get()
		return nil
	}

	// resolve all variables refered in the root string field
	childFieldPaths := rootField.dependsOn()
	for _, childFieldPath := range childFieldPaths {
		// recursive resolve variables in the child fields
		err := a.SafeResolve(childFieldPath, &seenPaths, fns...)
		if err != nil {
			return err
		}
	}

	// interpolate root string once all variables required are resolved
	rootField.interpolate(fns, a.memo)

	// record interpolated string in memo
	a.memo[path] = rootField.Get()
	return nil
}

// Interpolate all string fields in the config
func (a *accumulator) expand(fns ...LookupFunction) error {
	seenPaths := make([]string, 0)
	for path := range a.strings {
		err := a.SafeResolve(path, &seenPaths, fns...)
		if err != nil {
			return err
		}
	}
	return nil
}

type interpolate struct {
	fns []LookupFunction
}

func (m *interpolate) expand(v any) error {
	a := accumulator{}
	a.start(v)
	return a.expand(m.fns...)
}

func Interpolate(fns ...LookupFunction) bundle.Mutator {
	return &interpolate{fns: fns}
}

func (m *interpolate) Name() string {
	return "Interpolate"
}

func (m *interpolate) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	return nil, m.expand(&b.Config)
}
