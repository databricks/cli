package yamlloader

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/config"
	"gopkg.in/yaml.v3"
)

type loader struct {
	path string
}

func errorf(loc config.Location, format string, args ...interface{}) error {
	return fmt.Errorf("yaml (%s): %s", loc, fmt.Sprintf(format, args...))
}

func newLoader(path string) *loader {
	return &loader{
		path: path,
	}
}

func (d *loader) location(node *yaml.Node) config.Location {
	return config.Location{
		File:   d.path,
		Line:   node.Line,
		Column: node.Column,
	}
}

func (d *loader) load(node *yaml.Node) (config.Value, error) {
	loc := config.Location{
		File:   d.path,
		Line:   node.Line,
		Column: node.Column,
	}

	var value config.Value
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
		return config.NilValue, errorf(loc, "unknown node kind: %v", node.Kind)
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

func (d *loader) loadDocument(node *yaml.Node, loc config.Location) (config.Value, error) {
	return d.load(node.Content[0])
}

func (d *loader) loadSequence(node *yaml.Node, loc config.Location) (config.Value, error) {
	acc := make([]config.Value, len(node.Content))
	for i, n := range node.Content {
		v, err := d.load(n)
		if err != nil {
			return config.NilValue, err
		}

		acc[i] = v
	}

	return config.NewValue(acc, loc), nil
}

func (d *loader) loadMapping(node *yaml.Node, loc config.Location) (config.Value, error) {
	var merge *yaml.Node

	acc := make(map[string]config.Value)
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]

		// Assert that keys are strings
		if key.Kind != yaml.ScalarNode {
			return config.NilValue, errorf(loc, "key is not a scalar")
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
			return config.NilValue, errorf(loc, "invalid key tag: %v", st)
		}

		v, err := d.load(val)
		if err != nil {
			return config.NilValue, err
		}

		acc[key.Value] = v
	}

	if merge == nil {
		return config.NewValue(acc, loc), nil
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
		return config.NilValue, merr
	}

	// Build a sequence of values to merge.
	// The entries that we already accumulated have precedence.
	var seq []map[string]config.Value
	for _, n := range mnodes {
		v, err := d.load(n)
		if err != nil {
			return config.NilValue, err
		}
		m, ok := v.AsMap()
		if !ok {
			return config.NilValue, merr
		}
		seq = append(seq, m)
	}

	// Append the accumulated entries to the sequence.
	seq = append(seq, acc)
	out := make(map[string]config.Value)
	for _, m := range seq {
		for k, v := range m {
			out[k] = v
		}
	}

	return config.NewValue(out, loc), nil
}

func (d *loader) loadScalar(node *yaml.Node, loc config.Location) (config.Value, error) {
	st := node.ShortTag()
	switch st {
	case "!!str":
		return config.NewValue(node.Value, loc), nil
	case "!!bool":
		switch strings.ToLower(node.Value) {
		case "true":
			return config.NewValue(true, loc), nil
		case "false":
			return config.NewValue(false, loc), nil
		default:
			return config.NilValue, errorf(loc, "invalid bool value: %v", node.Value)
		}
	case "!!int":
		i64, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return config.NilValue, errorf(loc, "invalid int value: %v", node.Value)
		}
		// Use regular int type instead of int64 if possible.
		if i64 >= math.MinInt32 && i64 <= math.MaxInt32 {
			return config.NewValue(int(i64), loc), nil
		}
		return config.NewValue(i64, loc), nil
	case "!!float":
		f64, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return config.NilValue, errorf(loc, "invalid float value: %v", node.Value)
		}
		return config.NewValue(f64, loc), nil
	case "!!null":
		return config.NewValue(nil, loc), nil
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
				return config.NewValue(t, loc), nil
			}
		}
		return config.NilValue, errorf(loc, "invalid timestamp value: %v", node.Value)
	default:
		return config.NilValue, errorf(loc, "unknown tag: %v", st)
	}
}

func (d *loader) loadAlias(node *yaml.Node, loc config.Location) (config.Value, error) {
	return d.load(node.Alias)
}
