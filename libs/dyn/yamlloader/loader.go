package yamlloader

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/dyn"
	"gopkg.in/yaml.v3"
)

type loader struct {
	path string
}

func errorf(loc dyn.Location, format string, args ...interface{}) error {
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
		return dyn.NilValue, errorf(loc, "unknown node kind: %v", node.Kind)
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
			return dyn.NilValue, err
		}

		acc[i] = v
	}

	return dyn.NewValue(acc, loc), nil
}

func (d *loader) loadMapping(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	var merge *yaml.Node

	acc := dyn.NewMapping()
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]

		// Assert that keys are strings
		if key.Kind != yaml.ScalarNode {
			return dyn.NilValue, errorf(loc, "key is not a scalar")
		}

		st := key.ShortTag()
		switch st {
		case "!!str":
			// OK
		case "!!merge":
			if merge != nil {
				panic("merge node already set")
			}
			merge = val
			continue
		default:
			return dyn.NilValue, errorf(loc, "invalid key tag: %v", st)
		}

		k, err := d.load(key)
		if err != nil {
			return dyn.NilValue, err
		}

		v, err := d.load(val)
		if err != nil {
			return dyn.NilValue, err
		}

		acc.Set(k, v)
	}

	if merge == nil {
		return dyn.NewValue(acc, loc), nil
	}

	// Build location for the merge node.
	var mloc = d.location(merge)
	var merr = errorf(mloc, "map merge requires map or sequence of maps as the value")

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
			return dyn.NilValue, err
		}
		m, ok := v.AsMapping()
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

	return dyn.NewValue(out, loc), nil
}

func (d *loader) loadScalar(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	st := node.ShortTag()
	switch st {
	case "!!str":
		return dyn.NewValue(node.Value, loc), nil
	case "!!bool":
		switch strings.ToLower(node.Value) {
		case "true":
			return dyn.NewValue(true, loc), nil
		case "false":
			return dyn.NewValue(false, loc), nil
		default:
			return dyn.NilValue, errorf(loc, "invalid bool value: %v", node.Value)
		}
	case "!!int":
		i64, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return dyn.NilValue, errorf(loc, "invalid int value: %v", node.Value)
		}
		// Use regular int type instead of int64 if possible.
		if i64 >= math.MinInt32 && i64 <= math.MaxInt32 {
			return dyn.NewValue(int(i64), loc), nil
		}
		return dyn.NewValue(i64, loc), nil
	case "!!float":
		f64, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return dyn.NilValue, errorf(loc, "invalid float value: %v", node.Value)
		}
		return dyn.NewValue(f64, loc), nil
	case "!!null":
		return dyn.NewValue(nil, loc), nil
	case "!!timestamp":
		// Try a couple of layouts
		for _, layout := range []string{
			"2006-1-2T15:4:5.999999999Z07:00", // RCF3339Nano with short date fields.
			"2006-1-2t15:4:5.999999999Z07:00", // RFC3339Nano with short date fields and lower-case "t".
			"2006-1-2 15:4:5.999999999",       // space separated with no time zone
			"2006-1-2",                        // date only
		} {
			t, terr := time.Parse(layout, node.Value)
			if terr == nil {
				return dyn.NewValue(t, loc), nil
			}
		}
		return dyn.NilValue, errorf(loc, "invalid timestamp value: %v", node.Value)
	default:
		return dyn.NilValue, errorf(loc, "unknown tag: %v", st)
	}
}

func (d *loader) loadAlias(node *yaml.Node, loc dyn.Location) (dyn.Value, error) {
	return d.load(node.Alias)
}
