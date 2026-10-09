// Package structyaml writes typed values as YAML with a controlled key order.
package structyaml

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/structs/structvar"
	"go.yaml.in/yaml/v3"
)

// Pair is an entry of a [Map].
type Pair struct {
	Key   string
	Value any
}

// Map is a YAML mapping that keeps the order of its keys. Values are maps (Map or
// map[string]any, keys sorted), sequences ([]any or []string), strings, bools, ints, floats or nil.
type Map []Pair

// M returns a Map of the given alternating keys (strings) and values.
func M(kv ...any) Map {
	m := make(Map, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		m = append(m, Pair{Key: kv[i].(string), Value: kv[i+1]})
	}
	return m
}

// Add appends an entry.
func (m *Map) Add(key string, value any) {
	*m = append(*m, Pair{Key: key, Value: value})
}

// Order moves the given keys to the front, in the given order. The other keys keep
// their relative order.
func (m *Map) Order(keys ...string) Map {
	index := func(p Pair) int {
		if i := slices.Index(keys, p.Key); i >= 0 {
			return i
		}
		return len(keys)
	}
	slices.SortStableFunc(*m, func(a, b Pair) int { return cmp.Compare(index(a), index(b)) })
	return *m
}

// Struct returns the fields of the struct (or map) v that are set, except skip, as a
// Map with the keys sorted alphabetically, and so are the keys of maps and structs
// nested in maps. Sequence elements keep the field order of their type.
func Struct(v any, skip ...string) (Map, error) {
	x := structvar.NewView(v, nil, nil)
	if x.Kind() != structvar.KindMap {
		return nil, fmt.Errorf("expected map, got %s", x.Kind())
	}
	m := fromView(x, true).(Map)
	return slices.DeleteFunc(m, func(p Pair) bool { return slices.Contains(skip, p.Key) }), nil
}

// Value returns the typed value v as a value for a Map, in field order. Unset
// fields are dropped. It returns nil if v is unset.
func Value(v any) any {
	return fromView(structvar.NewView(v, nil, nil), false)
}

func fromView(x structvar.View, sorted bool) any {
	switch x.Kind() {
	case structvar.KindMap:
		var m Map
		for k, c := range x.MapItems() {
			m.Add(k, fromView(c, sorted))
		}
		if sorted {
			slices.SortStableFunc(m, func(a, b Pair) int { return cmp.Compare(a.Key, b.Key) })
		}
		return m
	case structvar.KindSequence:
		s := []any{} //nolint:gocritic // an empty sequence is written as [], not null
		for _, c := range x.Sequence() {
			s = append(s, fromView(c, false))
		}
		return s
	default:
		return x.AsAny()
	}
}

// Node returns v as a YAML node. Values of keys in styles (and everything below
// them) get that style.
func Node(v any, styles map[string]yaml.Style) *yaml.Node {
	return toNode(v, 0, styles)
}

func toNode(v any, style yaml.Style, styles map[string]yaml.Style) *yaml.Node {
	switch v := v.(type) {
	case Map:
		n := &yaml.Node{Kind: yaml.MappingNode, Style: style}
		for _, p := range v {
			s, ok := styles[p.Key]
			if !ok {
				s = style
			}
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: p.Key, Style: style}, toNode(p.Value, s, styles))
		}
		return n
	case map[string]any:
		m := make(Map, 0, len(v))
		for k, e := range v {
			m.Add(k, e)
		}
		slices.SortFunc(m, func(a, b Pair) int { return cmp.Compare(a.Key, b.Key) })
		return toNode(m, style, styles)
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode, Style: style}
		for _, e := range v {
			n.Content = append(n.Content, toNode(e, style, styles))
		}
		return n
	case []string:
		n := &yaml.Node{Kind: yaml.SequenceNode, Style: style}
		for _, e := range v {
			n.Content = append(n.Content, toNode(e, style, styles))
		}
		return n
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null", Style: style}
	case string:
		// A string that reads as another scalar (bool, number or empty) is quoted.
		if v == "" || v == "true" || v == "false" || isNumber(v) {
			style = yaml.DoubleQuotedStyle
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Style: style}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatBool(v), Style: style}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.Itoa(v), Style: style}
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatInt(v, 10), Style: style}
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v), Style: style}
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid type: %T", v))
	}
}

func isNumber(s string) bool {
	if _, err := strconv.ParseInt(s, 0, 64); err == nil {
		return true
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// Save writes v as YAML to path, creating parent directories. It fails if path exists
// and force is not set. See [Node] for styles.
func Save(path string, v any, force bool, styles map[string]yaml.Style) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", path)
		}
		if !force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", path)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	return enc.Encode(Node(v, styles))
}
