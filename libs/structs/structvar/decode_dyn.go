package structvar

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// DecodeDyn decodes the configuration tree v into dst, a pointer to a typed
// configuration value, and returns its references and locations (those of every
// value in v, including the ones merges accumulated).
//
// Values are converted to the field types like [DecodeYAML] converts them; the
// diagnostics explain values that were dropped.
func DecodeDyn(v dyn.Value, dst any) (*StructVar, diag.Diagnostics, error) {
	sv := &StructVar{Value: dst}
	rv := reflect.ValueOf(dst).Elem()
	if !v.IsValid() {
		rv.SetZero()
		return sv, nil, nil
	}
	var d decoder
	n, ok, err := d.decode(rv, dynSource{v}, nil)
	if err == nil && !ok {
		err = fmt.Errorf("expected a %s, found a %s", kindOf(rv.Type()), dynSource{v}.kind())
	}
	sv.Refs = n.refs(nil, nil)
	sv.Locations = n.locations()
	return sv, d.diags, err
}

// dynSource is a [source] backed by a [dyn.Value].
type dynSource struct {
	v dyn.Value
}

func (s dynSource) kind() Kind {
	switch s.v.Kind() {
	case dyn.KindMap:
		return KindMap
	case dyn.KindSequence:
		return KindSequence
	case dyn.KindString:
		return KindString
	case dyn.KindBool:
		return KindBool
	case dyn.KindInt:
		return KindInt
	case dyn.KindFloat:
		return KindFloat
	case dyn.KindTime:
		return KindTime
	case dyn.KindNil:
		return KindNil
	default:
		return KindInvalid
	}
}

func (s dynSource) locations() []diag.Location { return s.v.Locations() }
func (s dynSource) anchor() bool               { return s.v.IsAnchor() }

func (s dynSource) scalar() any {
	switch s.v.Kind() {
	case dyn.KindTime:
		return s.v.MustTime().String()
	case dyn.KindInt:
		// Like YAML values: int if it fits, int64 otherwise.
		i, _ := s.v.AsInt()
		return intValue(i)
	case dyn.KindFloat:
		// The decoder expects float64.
		f, _ := s.v.AsFloat()
		return f
	default:
		return s.v.AsAny()
	}
}

func (s dynSource) pairs() ([]sourcePair, error) {
	m, ok := s.v.AsMap()
	if !ok {
		return nil, nil
	}
	out := make([]sourcePair, 0, m.Len())
	for _, p := range m.Pairs() {
		out = append(out, sourcePair{key: p.Key.MustString(), keyLoc: p.Key.Location(), value: dynSource{p.Value}})
	}
	return out, nil
}

func (s dynSource) elems() ([]source, error) {
	seq, ok := s.v.AsSequence()
	if !ok {
		return nil, nil
	}
	out := make([]source, len(seq))
	for i, e := range seq {
		out[i] = dynSource{e}
	}
	return out, nil
}
