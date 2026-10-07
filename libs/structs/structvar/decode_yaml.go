package structvar

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/diag"
	"go.yaml.in/yaml/v3"
)

// LocationError is a YAML syntax error with the location it occurred at.
type LocationError struct {
	Loc     diag.Location
	Summary string
}

func (e *LocationError) Error() string {
	return fmt.Sprintf("yaml (%s): %s", e.Loc, e.Summary)
}

// yamlSource is a [source] backed by a YAML node. Aliases are followed, merge keys
// are applied, and scalars are resolved by their tag.
type yamlSource struct {
	file string
	node *yaml.Node

	// The node the value comes from, after following aliases.
	target *yaml.Node

	// Aliases followed on the way from the document root to this value, to detect
	// cyclic anchors.
	aliases *aliasChain

	// Resolved scalar.
	k Kind
	v any
}

// ParseYAML parses the YAML document in r. It returns nil for an empty document.
func ParseYAML(r io.Reader) (*yaml.Node, error) {
	var doc yaml.Node
	err := yaml.NewDecoder(r).Decode(&doc)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.Content[0], nil
}

// aliasChain is an immutable list of alias nodes.
type aliasChain struct {
	node *yaml.Node
	prev *aliasChain
}

func (c *aliasChain) contains(n *yaml.Node) bool {
	for ; c != nil; c = c.prev {
		if c.node == n {
			return true
		}
	}
	return false
}

func newYAMLSource(file string, node *yaml.Node, aliases *aliasChain) (*yamlSource, error) {
	s := &yamlSource{file: file, node: node, target: node, aliases: aliases}
	loc := s.loc(node)
	for s.target.Kind == yaml.AliasNode {
		// The same alias may be reached again through another path, which is not a
		// cycle; only an alias inside its own expansion is.
		if s.aliases.contains(s.target) {
			return nil, yamlErrorf(loc, "cyclic reference to anchor %q", s.target.Value)
		}
		s.aliases = &aliasChain{node: s.target, prev: s.aliases}
		s.target = s.target.Alias
	}

	switch s.target.Kind {
	case yaml.MappingNode:
		s.k = KindMap
	case yaml.SequenceNode:
		s.k = KindSequence
	case yaml.ScalarNode:
		if err := s.resolveScalar(loc); err != nil {
			return nil, err
		}
	default:
		return nil, yamlErrorf(loc, "unknown node kind: %v", s.target.Kind)
	}
	return s, nil
}

func yamlErrorf(loc diag.Location, format string, args ...any) error {
	return fmt.Errorf("yaml (%s): %s", loc, fmt.Sprintf(format, args...))
}

func (s *yamlSource) loc(n *yaml.Node) diag.Location {
	return diag.Location{File: s.file, Line: n.Line, Column: n.Column}
}

func intValue(i64 int64) any {
	// Use regular int type instead of int64 if possible.
	if i64 >= math.MinInt32 && i64 <= math.MaxInt32 {
		return int(i64)
	}
	return i64
}

func (s *yamlSource) resolveScalar(loc diag.Location) error {
	n := s.target
	switch st := n.ShortTag(); st {
	case "!!str":
		s.k, s.v = KindString, n.Value
	case "!!bool":
		switch strings.ToLower(n.Value) {
		case "true":
			s.k, s.v = KindBool, true
		case "false":
			s.k, s.v = KindBool, false
		default:
			return yamlErrorf(loc, "invalid bool value: %v", n.Value)
		}
	case "!!int":
		// Try to parse the integer value in base 10. Trim leading zeros to avoid
		// octal parsing of the "0" prefix (YAML 1.2 spec example 2.19).
		i64, err := strconv.ParseInt(strings.TrimLeft(n.Value, "0"), 10, 64)
		if err != nil {
			// Let ParseInt figure out the base.
			i64, err = strconv.ParseInt(n.Value, 0, 64)
		}
		if err != nil {
			return yamlErrorf(loc, "invalid int value: %v", n.Value)
		}
		s.k, s.v = KindInt, intValue(i64)
	case "!!float":
		f64, err := strconv.ParseFloat(n.Value, 64)
		if err == nil {
			s.k, s.v = KindFloat, f64
			break
		}
		// Deal with infinity prefixes.
		v := strings.ToLower(n.Value)
		switch {
		case strings.HasPrefix(v, "+"):
			v = strings.TrimPrefix(v, "+")
			f64 = math.Inf(1)
		case strings.HasPrefix(v, "-"):
			v = strings.TrimPrefix(v, "-")
			f64 = math.Inf(-1)
		default:
			f64 = math.Inf(1)
		}
		switch v {
		case ".inf":
			s.k, s.v = KindFloat, f64
		case ".nan":
			s.k, s.v = KindFloat, math.NaN()
		default:
			return yamlErrorf(loc, "invalid float value: %v", n.Value)
		}
	case "!!null":
		s.k = KindNil
	case "!!timestamp":
		if !isTimestamp(n.Value) {
			return yamlErrorf(loc, "invalid timestamp value: %v", n.Value)
		}
		// Timestamps are kept as the string they were written as.
		s.k, s.v = KindTime, n.Value
	default:
		return yamlErrorf(loc, "unknown tag: %v", st)
	}
	return nil
}

func isTimestamp(s string) bool {
	for _, layout := range []string{
		"2006-1-2T15:4:5.999999999Z07:00", // RCF3339Nano with short date fields.
		"2006-1-2t15:4:5.999999999Z07:00", // RFC3339Nano with short date fields and lower-case "t".
		"2006-1-2 15:4:5.999999999",       // space separated with no time zone
		"2006-1-2",                        // date only
	} {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

func (s *yamlSource) kind() Kind {
	return s.k
}

func (s *yamlSource) locations() []diag.Location {
	// An alias has the location of the anchored value.
	return []diag.Location{s.loc(s.target)}
}

func (s *yamlSource) anchor() bool {
	return s.node.Anchor != "" || s.target.Anchor != ""
}

func (s *yamlSource) scalar() any {
	return s.v
}

func (s *yamlSource) child(n *yaml.Node) (source, error) {
	return newYAMLSource(s.file, n, s.aliases)
}

func (s *yamlSource) pairs() ([]sourcePair, error) {
	node := s.target
	loc := s.loc(node)
	var merge *yaml.Node
	var acc pairSet
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]

		if key.Kind != yaml.ScalarNode {
			return nil, yamlErrorf(loc, "key is not a scalar")
		}

		switch st := key.ShortTag(); st {
		case "!!str":
		case "!!null":
			// A literal unquoted "null" is treated as a null value by the YAML parser.
			// However, when used as a key, it is treated as the string "null".
		case "!!merge":
			if merge != nil {
				// The YAML merge key spec allows a single '<<' key per mapping.
				return nil, &LocationError{
					Loc:     s.loc(key),
					Summary: "duplicate YAML merge key ('<<') is not allowed; to merge multiple maps, use a sequence: '<<: [*anchor1, *anchor2]'",
				}
			}
			merge = val
			continue
		default:
			return nil, yamlErrorf(loc, "invalid key tag: %v", st)
		}

		v, err := s.child(val)
		if err != nil {
			return nil, err
		}
		acc.set(sourcePair{key: key.Value, keyLocs: []diag.Location{s.loc(key)}, value: v})
	}

	if merge == nil {
		return acc.pairs, nil
	}

	mloc := s.loc(merge)
	merr := yamlErrorf(mloc, "map merge requires map or sequence of maps as the value")

	var mnodes []*yaml.Node
	switch merge.Kind {
	case yaml.SequenceNode:
		mnodes = merge.Content
	case yaml.AliasNode:
		mnodes = []*yaml.Node{merge}
	default:
		return nil, merr
	}

	// Merged maps come first; the entries of this mapping take precedence.
	var out pairSet
	for _, n := range mnodes {
		ms, err := s.child(n)
		if err != nil {
			return nil, err
		}
		if ms.kind() != KindMap {
			return nil, merr
		}
		mps, err := ms.pairs()
		if err != nil {
			return nil, err
		}
		for _, p := range mps {
			out.set(p)
		}
	}
	for _, p := range acc.pairs {
		out.set(p)
	}
	return out.pairs, nil
}

// setPair sets the pair in pairs: an existing key keeps its position and key location.
// pairSet is a list of pairs with unique keys, in the order the keys were first set.
type pairSet struct {
	pairs []sourcePair
	index map[string]int
}

// set adds p, or replaces the value of the pair with the same key (keeping its key
// location and position).
func (s *pairSet) set(p sourcePair) {
	if i, ok := s.index[p.key]; ok {
		s.pairs[i].value = p.value
		return
	}
	if s.index == nil {
		s.index = map[string]int{}
	}
	s.index[p.key] = len(s.pairs)
	s.pairs = append(s.pairs, p)
}

func (s *yamlSource) elems() ([]source, error) {
	var out []source
	for _, n := range s.target.Content {
		v, err := s.child(n)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
