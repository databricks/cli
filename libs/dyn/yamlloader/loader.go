package yamlloader

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"gopkg.in/yaml.v3"
)

type loader struct {
	path string
}

func errorf(loc dyn.Location, format string, args ...any) error {
	return fmt.Errorf("yaml (%s): %s", loc, fmt.Sprintf(format, args...))
}

func newLoader(path string) *loader {
	return &loader{
		path: path,
	}
}

func (d *loader) location(node *yaml.Node) dyn.Location {
	return dyn.Location{
		File:   d.path,
		Line:   node.Line,
		Column: node.Column,
	}
}

func (d *loader) load(node *yaml.Node) (dyn.Value, error) {
	loc := dyn.Location{
		File:   d.path,
		Line:   node.Line,
		Column: node.Column,
	}

	var value dyn.Value
	var err error

	switch node.Kind {
	case yaml.DocumentNode:
		value, err = d.loadDocument(node, loc)
	case yaml.SequenceNode:
		value, err = d.loadSequence(node, loc)
	case yaml.MappingNode:
		value, err = d.loadMapping(node, loc)
	case yaml.ScalarNode:
		value, err = d.loadScalar(node, loc)
	case yaml.AliasNode:
		value, err = d.loadAlias(node, loc)
	default:
		return dyn.InvalidValue, errorf(loc, "unknown node kind: %v", node.Kind)
	}

	if err != nil {
		return value, err
	}

	// Mark value as anchor if needed.
	// If this node doesn't map to a type, we don't need to warn about it.
	if node.Anchor != "" {
		value = value.MarkAnchor()
	}

	return value, nil
}

func (d *loader) loadDocument(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	return d.load(node.Content[0])
}

func (d *loader) loadSequence(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	acc := make([]dyn.Value, len(node.Content))
	for i, n := range node.Content {
		v, err := d.load(n)
		if err != nil {
			return dyn.InvalidValue, err
		}

		acc[i] = v
	}

	return dyn.NewValue(acc, []dyn.Location{loc}), nil
}

func (d *loader) loadMapping(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	var merge *yaml.Node

	acc := dyn.NewMapping()
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]

		// Assert that keys are strings
		if key.Kind != yaml.ScalarNode {
			return dyn.InvalidValue, errorf(loc, "key is not a scalar")
		}

		st := key.ShortTag()
		switch st {
		case "!!str":
			// OK
		case "!!null":
			// A literal unquoted "null" is treated as a null value by the YAML parser.
			// However, when used as a key, it is treated as the string "null".
		case "!!merge":
			if merge != nil {
				panic("merge node already set")
			}
			merge = val
			continue
		default:
			return dyn.InvalidValue, errorf(loc, "invalid key tag: %v", st)
		}

		loc := []dyn.Location{{
			File:   d.path,
			Line:   key.Line,
			Column: key.Column,
		}}

		v, err := d.load(val)
		if err != nil {
			return dyn.InvalidValue, err
		}

		acc.SetLoc(key.Value, loc, v)
	}

	if merge == nil {
		return dyn.NewValue(acc, []dyn.Location{loc}), nil
	}

	// Build location for the merge node.
	mloc := d.location(merge)
	merr := errorf(mloc, "map merge requires map or sequence of maps as the value")

	// Flatten the merge node into a slice of nodes.
	// It can be either a single node or a sequence of nodes.
	var mnodes []*yaml.Node
	switch merge.Kind {
	case yaml.SequenceNode:
		mnodes = merge.Content
	case yaml.AliasNode:
		mnodes = []*yaml.Node{merge}
	default:
		return dyn.NilValue, merr
	}

	// Build a sequence of values to merge.
	// The entries that we already accumulated have precedence.
	var seq []dyn.Mapping
	for _, n := range mnodes {
		v, err := d.load(n)
		if err != nil {
			return dyn.InvalidValue, err
		}
		m, ok := v.AsMap()
		if !ok {
			return dyn.NilValue, merr
		}
		seq = append(seq, m)
	}

	// Append the accumulated entries to the sequence.
	seq = append(seq, acc)
	out := dyn.NewMapping()
	for _, m := range seq {
		out.Merge(m)
	}

	return dyn.NewValue(out, []dyn.Location{loc}), nil
}

func newIntValue(i64 int64, loc dyn.Location) dyn.Value {
	// Use regular int type instead of int64 if possible.
	if i64 >= math.MinInt32 && i64 <= math.MaxInt32 {
		return dyn.NewValue(int(i64), []dyn.Location{loc})
	}
	return dyn.NewValue(i64, []dyn.Location{loc})
}

func (d *loader) loadScalar(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	st := node.ShortTag()
	switch st {
	case "!!str":
		return dyn.NewValue(node.Value, []dyn.Location{loc}), nil
	case "!!bool":
		switch strings.ToLower(node.Value) {
		case "true":
			return dyn.NewValue(true, []dyn.Location{loc}), nil
		case "false":
			return dyn.NewValue(false, []dyn.Location{loc}), nil
		default:
			return dyn.InvalidValue, errorf(loc, "invalid bool value: %v", node.Value)
		}
	case "!!int":
		// Try to parse the an integer value in base 10.
		// We trim leading zeros to avoid octal parsing of the "0" prefix.
		// See "testdata/spec_example_2.19.yml" for background.
		i64, err := strconv.ParseInt(strings.TrimLeft(node.Value, "0"), 10, 64)
		if err == nil {
			return newIntValue(i64, loc), nil
		}
		// Let the [ParseInt] function figure out the base.
		i64, err = strconv.ParseInt(node.Value, 0, 64)
		if err == nil {
			return newIntValue(i64, loc), nil
		}
		return dyn.InvalidValue, errorf(loc, "invalid int value: %v", node.Value)
	case "!!float":
		f64, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			// Deal with infinity prefixes.
			v := strings.ToLower(node.Value)
			switch {
			case strings.HasPrefix(v, "+"):
				v = strings.TrimPrefix(v, "+")
				f64 = math.Inf(1)
			case strings.HasPrefix(v, "-"):
				v = strings.TrimPrefix(v, "-")
				f64 = math.Inf(-1)
			default:
				// No prefix.
				f64 = math.Inf(1)
			}

			// Deal with infinity and NaN values.
			switch v {
			case ".inf":
				return dyn.NewValue(f64, []dyn.Location{loc}), nil
			case ".nan":
				return dyn.NewValue(math.NaN(), []dyn.Location{loc}), nil
			}

			return dyn.InvalidValue, errorf(loc, "invalid float value: %v", node.Value)
		}
		return dyn.NewValue(f64, []dyn.Location{loc}), nil
	case "!!null":
		return dyn.NewValue(nil, []dyn.Location{loc}), nil
	case "!!timestamp":
		t, err := dyn.NewTime(node.Value)
		if err == nil {
			return dyn.NewValue(t, []dyn.Location{loc}), nil
		}
		return dyn.InvalidValue, errorf(loc, "invalid timestamp value: %v", node.Value)
	default:
		return dyn.InvalidValue, errorf(loc, "unknown tag: %v", st)
	}
}

func (d *loader) loadAlias(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	return d.load(node.Alias)
}
