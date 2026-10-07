package structvar

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"go.yaml.in/yaml/v3"
)

// source is a configuration tree to decode from (a parsed YAML document or a view).
type source interface {
	kind() Kind
	locations() []diag.Location
	// anchor reports whether the value is a YAML anchor (or an alias of one).
	anchor() bool
	// scalar returns the value of a string, bool, int (int or int64), float or time (string).
	scalar() any
	pairs() ([]sourcePair, error)
	elems() ([]source, error)
}

type sourcePair struct {
	key     string
	keyLocs []diag.Location
	value   source
}

// DecodeYAML decodes the YAML document in r (read from file) into dst, a pointer to a
// typed configuration value, and returns its references and locations.
//
// Values are converted to the field types the way configuration is normalized: scalars
// are converted where possible, pure variable references are recorded in the returned
// references for fields that cannot hold a string, and values that cannot be converted
// or fields that do not exist are dropped with a warning. An error is returned for
// invalid YAML.
func DecodeYAML(file string, r io.Reader, dst any) (*StructVar, diag.Diagnostics, error) {
	node, err := ParseYAML(r)
	if err != nil {
		return nil, nil, err
	}
	return DecodeYAMLNode(file, node, dst, nil)
}

// LocationMapper returns the locations to record for the value at path, given the
// locations it has in the YAML document.
type LocationMapper func(path *structpath.PathNode, locs []diag.Location) []diag.Location

// DecodeYAMLNode is [DecodeYAML] for a parsed YAML document (nil if it is empty).
// If mapLocations is not nil, it is applied to the location of every value.
func DecodeYAMLNode(file string, node *yaml.Node, dst any, mapLocations LocationMapper) (*StructVar, diag.Diagnostics, error) {
	sv := &StructVar{Value: dst}
	v := reflect.ValueOf(dst).Elem()
	if node == nil {
		// An empty document is null.
		v.SetZero()
		return sv, nil, nil
	}
	src, err := newYAMLSource(file, node, nil)
	if err != nil {
		return nil, nil, err
	}
	// Report YAML errors anywhere in the document, also in values that are dropped.
	if err := validate(src); err != nil {
		return nil, nil, err
	}
	var d decoder
	var s source = src
	if mapLocations != nil {
		s = mappedSource{source: src, fn: mapLocations}
	}
	n, ok, err := d.decode(v, s, nil)
	if err == nil && !ok {
		// The value could not be converted to dst at all (e.g. a list for a struct).
		err = fmt.Errorf("expected a %s, found a %s", kindOf(v.Type()), src.kind())
	}
	sv.Refs = n.refs(nil, nil)
	sv.Locations = n.locations()
	return sv, d.diags, err
}

// Assign sets the value described by src at path, replacing what was there, with the
// references and locations of src. The value is converted to the type at path like
// [DecodeYAML] converts values; the diagnostics explain values that were dropped.
func (sv *StructVar) Assign(path *structpath.PathNode, src View) (diag.Diagnostics, error) {
	var d decoder
	var n *dnode
	err := update(reflect.ValueOf(sv.Value), components(path), func(dst reflect.Value) (bool, error) {
		var ok bool
		var err error
		n, ok, err = d.decode(dst, viewSource{src}, path)
		if !ok {
			dst.SetZero()
			return false, err
		}
		return n.isZeroValue(dst), err
	})
	if err != nil {
		return d.diags, err
	}
	sv.Refs = n.refs(path, withoutRefs(sv.Refs, path))
	sv.Locations = sv.Locations.With(path, n.locations())
	return d.diags, nil
}

func validate(s source) error {
	switch s.kind() {
	case KindMap:
		pairs, err := s.pairs()
		if err != nil {
			return err
		}
		for _, p := range pairs {
			if err := validate(p.value); err != nil {
				return err
			}
		}
	case KindSequence:
		elems, err := s.elems()
		if err != nil {
			return err
		}
		for _, e := range elems {
			if err := validate(e); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

type decoder struct {
	diags diag.Diagnostics
}

// dnode is the decoded shape of a value: what the typed value does not hold.
type dnode struct {
	kind  Kind
	locs  []diag.Location
	ref   string
	keys  map[string]*dnode
	elems []*dnode
}

// locations returns the locations of the decoded value.
func (n *dnode) locations() *Locations {
	if n == nil {
		return nil
	}
	out := &Locations{locs: n.locs}
	for k, c := range n.keys {
		if out.keys == nil {
			out.keys = make(map[string]*Locations, len(n.keys))
		}
		out.keys[k] = c.locations()
	}
	for _, c := range n.elems {
		out.elems = append(out.elems, c.locations())
	}
	return out
}

// refs returns refs (copied) plus the references of the decoded value at path.
func (n *dnode) refs(path *structpath.PathNode, refs map[string]string) map[string]string {
	var out map[string]string
	n.collectRefs(path, func(p *structpath.PathNode, ref string) {
		if out == nil {
			out = maps.Clone(refs)
			if out == nil {
				out = map[string]string{}
			}
		}
		out[p.String()] = ref
	})
	if out == nil {
		return refs
	}
	return out
}

func (n *dnode) collectRefs(path *structpath.PathNode, fn func(*structpath.PathNode, string)) {
	if n == nil {
		return
	}
	if n.ref != "" {
		fn(path, n.ref)
	}
	for k, c := range n.keys {
		c.collectRefs(structpath.NewStringKey(path, k), fn)
	}
	for i, c := range n.elems {
		c.collectRefs(structpath.NewIndex(path, i), fn)
	}
}

func (d *decoder) warn(summary string, src source, path *structpath.PathNode) {
	d.diags = d.diags.Append(diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   summary,
		Locations: []diag.Location{firstLocation(src)},
		Paths:     []*structpath.PathNode{path},
	})
}

func firstLocation(src source) diag.Location {
	locs := src.locations()
	if len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

func (d *decoder) typeMismatch(expected Kind, src source, path *structpath.PathNode) {
	d.warn(fmt.Sprintf("expected %s, found %s", expected, src.kind()), src, path)
}

func (d *decoder) nullWarning(expected Kind, src source, path *structpath.PathNode) {
	d.warn(fmt.Sprintf("expected a %s value, found null", expected), src, path)
}

func pureRefOf(src source) string {
	if src.kind() != KindString {
		return ""
	}
	s, _ := src.scalar().(string)
	if IsPureVariableReference(s) {
		return s
	}
	return ""
}

// decode decodes src into dst (settable) and returns its meta. It reports false if the
// value cannot be decoded and is dropped (the diagnostics say why).
func (d *decoder) decode(dst reflect.Value, src source, path *structpath.PathNode) (*dnode, bool, error) {
	typ := dst.Type()
	if typ.Kind() == reflect.Pointer {
		if src.kind() == KindNil {
			n, ok, err := d.decode(reflect.New(typ.Elem()).Elem(), src, path)
			if ok {
				dst.SetZero()
			}
			return n, ok, err
		}
		p := reflect.New(typ.Elem())
		n, ok, err := d.decode(p.Elem(), src, path)
		if ok {
			dst.Set(p)
		}
		return n, ok, err
	}

	n := &dnode{kind: src.kind(), locs: src.locations()}
	ref := pureRefOf(src)

	switch typ.Kind() {
	case reflect.Struct:
		if IsSDKNativeType(typ) {
			s, ok := d.decodeString(src, path)
			if !ok {
				return nil, false, nil
			}
			n.kind = KindString
			if ref != "" {
				n.ref = ref
				dst.SetZero()
				return n, true, nil
			}
			buf, err := json.Marshal(s)
			if err != nil {
				return nil, false, err
			}
			return n, true, json.Unmarshal(buf, dst.Addr().Interface())
		}
		return d.decodeStruct(dst, src, path, n, ref)
	case reflect.Map:
		switch src.kind() {
		case KindMap:
			pairs, err := src.pairs()
			if err != nil {
				return nil, false, err
			}
			out := reflect.MakeMapWithSize(typ, len(pairs))
			n.keys = make(map[string]*dnode, len(pairs))
			for _, p := range pairs {
				e := reflect.New(typ.Elem()).Elem()
				en, ok, err := d.decode(e, p.value, structpath.NewStringKey(path, p.key))
				if err != nil {
					return nil, false, err
				}
				if !ok {
					continue
				}
				out.SetMapIndex(reflect.ValueOf(p.key).Convert(typ.Key()), e)
				n.keys[p.key] = en
			}
			dst.Set(out)
			return n, true, nil
		case KindNil:
			dst.SetZero()
			return n, true, nil
		default:
		}
		if ref != "" {
			n.ref = ref
			dst.SetZero()
			return n, true, nil
		}
		d.typeMismatch(KindMap, src, path)
		return nil, false, nil
	case reflect.Slice:
		switch src.kind() {
		case KindSequence:
			elems, err := src.elems()
			if err != nil {
				return nil, false, err
			}
			out := reflect.MakeSlice(typ, 0, len(elems))
			for _, ev := range elems {
				e := reflect.New(typ.Elem()).Elem()
				en, ok, err := d.decode(e, ev, structpath.NewIndex(path, out.Len()))
				if err != nil {
					return nil, false, err
				}
				if !ok {
					continue
				}
				out = reflect.Append(out, e)
				n.elems = append(n.elems, en)
			}
			dst.Set(out)
			return n, true, nil
		case KindNil:
			dst.SetZero()
			return n, true, nil
		default:
		}
		if ref != "" {
			n.ref = ref
			dst.SetZero()
			return n, true, nil
		}
		d.typeMismatch(KindSequence, src, path)
		return nil, false, nil
	case reflect.String:
		s, ok := d.decodeString(src, path)
		if !ok {
			return nil, false, nil
		}
		n.kind = KindString
		dst.SetString(s)
		return n, true, nil
	case reflect.Bool:
		return d.decodeBool(dst, src, path, n, ref)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return d.decodeInt(dst, src, path, n, ref)
	case reflect.Float32, reflect.Float64:
		return d.decodeFloat(dst, src, path, n, ref)
	case reflect.Interface:
		v, err := d.decodeAny(src, n)
		if err != nil {
			return nil, false, err
		}
		if v == nil {
			dst.SetZero()
		} else {
			dst.Set(reflect.ValueOf(v))
		}
		return n, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported type: %s", typ.Kind())
	}
}

// isAnchorContainer reports whether v is a YAML anchor or a non-empty sequence or map
// composed entirely of anchor containers. Anchors define reusable blocks and must not
// trigger "unknown field" warnings, including when nested inside a container.
func isAnchorContainer(v source) bool {
	if v.anchor() {
		return true
	}
	var elements []source
	switch v.kind() {
	case KindSequence:
		elements, _ = v.elems()
	case KindMap:
		pairs, _ := v.pairs()
		for _, p := range pairs {
			elements = append(elements, p.value)
		}
	default:
		return false
	}
	if len(elements) == 0 {
		return false
	}
	for _, e := range elements {
		if !isAnchorContainer(e) {
			return false
		}
	}
	return true
}

func (d *decoder) decodeStruct(dst reflect.Value, src source, path *structpath.PathNode, n *dnode, ref string) (*dnode, bool, error) {
	switch src.kind() {
	case KindMap:
	case KindNil:
		dst.SetZero()
		return n, true, nil
	default:
		if ref != "" {
			n.ref = ref
			dst.SetZero()
			return n, true, nil
		}
		d.typeMismatch(KindMap, src, path)
		return nil, false, nil
	}

	pairs, err := src.pairs()
	if err != nil {
		return nil, false, err
	}

	dst.SetZero()
	info := GetStructInfo(dst.Type())
	n.keys = make(map[string]*dnode, len(pairs))
	for _, p := range pairs {
		index, ok := info.Fields[p.key]
		if !ok {
			if isAnchorContainer(p.value) {
				continue
			}
			// Special case: provide a more helpful message for "valueFrom" vs "value_from".
			if _, hasValueFrom := info.Fields["value_from"]; p.key == "valueFrom" && hasValueFrom {
				d.diags = d.diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "Use 'value_from' instead of 'valueFrom'",
					Detail:    "The field 'valueFrom' should be 'value_from' (snake_case). The 'valueFrom' field will be ignored.",
					Locations: p.keyLocs,
					Paths:     []*structpath.PathNode{path},
				})
				continue
			}
			d.diags = d.diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   "unknown field: " + p.key,
				Locations: p.keyLocs,
				Paths:     []*structpath.PathNode{path},
			})
			continue
		}

		// Decode into a fresh value so a dropped value leaves the field (and any
		// embedded struct pointers on the way to it) untouched.
		ft := dst.Type().FieldByIndex(index).Type
		fv := reflect.New(ft).Elem()
		fn, ok, err := d.decode(fv, p.value, structpath.NewStringKey(path, p.key))
		if err != nil {
			return nil, false, err
		}
		if !ok {
			continue
		}
		GetOrNewFieldByIndex(dst, index).Set(fv)
		n.keys[p.key] = fn

		// An explicitly set zero value must still serialize: add it to ForceSendFields.
		if fn.isZeroValue(fv) {
			addForceSend(dst, &info, p.key)
		}
	}
	return n, true, nil
}

// isZeroValue mirrors dyn.Value.IsZero for the decoded value: nil, an empty map or
// sequence, or a zero scalar (a pure reference is not zero).
func (n *dnode) isZeroValue(v reflect.Value) bool {
	switch n.kind {
	case KindNil:
		return true
	case KindMap:
		return len(n.keys) == 0
	case KindSequence:
		return len(n.elems) == 0
	default:
	}
	if n.ref != "" {
		return false
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		// SDK native types are strings in the configuration tree.
		s, ok := sdkNativeString(v, includeZero)
		return !ok || s == ""
	}
	return v.IsZero()
}

func (d *decoder) decodeString(src source, path *structpath.PathNode) (string, bool) {
	switch src.kind() {
	case KindString, KindTime:
		return src.scalar().(string), true
	case KindBool:
		return strconv.FormatBool(src.scalar().(bool)), true
	case KindInt:
		return strconv.FormatInt(toInt64(src.scalar()), 10), true
	case KindFloat:
		return strconv.FormatFloat(src.scalar().(float64), 'f', -1, 64), true
	case KindNil:
		d.nullWarning(KindString, src, path)
		return "", false
	default:
		d.typeMismatch(KindString, src, path)
		return "", false
	}
}

func toInt64(v any) int64 {
	switch v := v.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	default:
		panic(fmt.Sprintf("unexpected int type %T", v))
	}
}

func (d *decoder) decodeBool(dst reflect.Value, src source, path *structpath.PathNode, n *dnode, ref string) (*dnode, bool, error) {
	switch src.kind() {
	case KindBool:
		dst.SetBool(src.scalar().(bool))
		return n, true, nil
	case KindString:
		// See https://yaml.org/type/bool.html.
		switch src.scalar().(string) {
		case "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
			dst.SetBool(true)
		case "false", "False", "FALSE", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
			dst.SetBool(false)
		default:
			if ref != "" {
				n.ref = ref
				dst.SetZero()
				return n, true, nil
			}
			d.typeMismatch(KindBool, src, path)
			return nil, false, nil
		}
		n.kind = KindBool
		return n, true, nil
	case KindNil:
		d.nullWarning(KindBool, src, path)
		return nil, false, nil
	default:
		d.typeMismatch(KindBool, src, path)
		return nil, false, nil
	}
}

func (d *decoder) decodeInt(dst reflect.Value, src source, path *structpath.PathNode, n *dnode, ref string) (*dnode, bool, error) {
	switch src.kind() {
	case KindInt:
		dst.SetInt(toInt64(src.scalar()))
		return n, true, nil
	case KindFloat:
		f := src.scalar().(float64)
		out := int64(f)
		if f != float64(out) {
			d.warn(fmt.Sprintf(`cannot accurately represent "%g" as integer due to precision loss`, f), src, path)
			return nil, false, nil
		}
		dst.SetInt(out)
		n.kind = KindInt
		return n, true, nil
	case KindString:
		s := src.scalar().(string)
		out, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			if ref != "" {
				n.ref = ref
				dst.SetZero()
				return n, true, nil
			}
			d.warn(fmt.Sprintf("cannot parse %q as an integer", s), src, path)
			return nil, false, nil
		}
		dst.SetInt(out)
		n.kind = KindInt
		return n, true, nil
	case KindNil:
		d.nullWarning(KindInt, src, path)
		return nil, false, nil
	default:
		d.typeMismatch(KindInt, src, path)
		return nil, false, nil
	}
}

func (d *decoder) decodeFloat(dst reflect.Value, src source, path *structpath.PathNode, n *dnode, ref string) (*dnode, bool, error) {
	switch src.kind() {
	case KindFloat:
		dst.SetFloat(src.scalar().(float64))
		return n, true, nil
	case KindInt:
		i := toInt64(src.scalar())
		out := float64(i)
		if i != int64(out) {
			d.warn(fmt.Sprintf(`cannot accurately represent "%d" as floating point number due to precision loss`, i), src, path)
			return nil, false, nil
		}
		dst.SetFloat(out)
		n.kind = KindFloat
		return n, true, nil
	case KindString:
		s := src.scalar().(string)
		out, err := strconv.ParseFloat(s, 64)
		if err != nil {
			if ref != "" {
				n.ref = ref
				dst.SetZero()
				return n, true, nil
			}
			d.warn(fmt.Sprintf("cannot parse %q as a floating point number", s), src, path)
			return nil, false, nil
		}
		dst.SetFloat(out)
		n.kind = KindFloat
		return n, true, nil
	case KindNil:
		d.nullWarning(KindFloat, src, path)
		return nil, false, nil
	default:
		d.typeMismatch(KindFloat, src, path)
		return nil, false, nil
	}
}

// decodeAny returns the generic Go value for src (maps, slices, scalars) and fills in
// the meta of everything below n. Timestamps become the string they were written as.
func (d *decoder) decodeAny(src source, n *dnode) (any, error) {
	switch src.kind() {
	case KindMap:
		pairs, err := src.pairs()
		if err != nil {
			return nil, err
		}
		out := make(map[string]any, len(pairs))
		n.keys = make(map[string]*dnode, len(pairs))
		for _, p := range pairs {
			cn := &dnode{kind: p.value.kind(), locs: p.value.locations()}
			v, err := d.decodeAny(p.value, cn)
			if err != nil {
				return nil, err
			}
			out[p.key] = v
			n.keys[p.key] = cn
		}
		return out, nil
	case KindSequence:
		elems, err := src.elems()
		if err != nil {
			return nil, err
		}
		out := make([]any, len(elems))
		for i, e := range elems {
			cn := &dnode{kind: e.kind(), locs: e.locations()}
			v, err := d.decodeAny(e, cn)
			if err != nil {
				return nil, err
			}
			out[i] = v
			n.elems = append(n.elems, cn)
		}
		return out, nil
	case KindTime:
		n.kind = KindString
		return src.scalar(), nil
	case KindNil:
		return nil, nil
	default:
		return src.scalar(), nil
	}
}

// viewSource is a [source] backed by a view.
type viewSource struct {
	v View
}

func (s viewSource) kind() Kind                 { return s.v.Kind() }
func (s viewSource) locations() []diag.Location { return s.v.Locations() }
func (s viewSource) anchor() bool               { return false }

func (s viewSource) scalar() any {
	if str, ok := s.v.AsString(); ok {
		return str
	}
	return s.v.AsAny()
}

func (s viewSource) pairs() ([]sourcePair, error) {
	var out []sourcePair
	for k, c := range s.v.MapItems() {
		out = append(out, sourcePair{key: k, value: viewSource{c}})
	}
	return out, nil
}

func (s viewSource) elems() ([]source, error) {
	var out []source
	for _, c := range s.v.Sequence() {
		out = append(out, viewSource{c})
	}
	return out, nil
}

// mappedSource is a [source] whose locations are mapped by fn.
type mappedSource struct {
	source
	path *structpath.PathNode
	fn   LocationMapper
}

func (s mappedSource) locations() []diag.Location {
	return s.fn(s.path, s.source.locations())
}

func (s mappedSource) pairs() ([]sourcePair, error) {
	pairs, err := s.source.pairs()
	for i := range pairs {
		pairs[i].value = mappedSource{source: pairs[i].value, path: structpath.NewStringKey(s.path, pairs[i].key), fn: s.fn}
	}
	return pairs, err
}

func (s mappedSource) elems() ([]source, error) {
	elems, err := s.source.elems()
	for i := range elems {
		elems[i] = mappedSource{source: elems[i], path: structpath.NewIndex(s.path, i), fn: s.fn}
	}
	return elems, err
}

// kindOf returns the kind of configuration tree values of type t.
func kindOf(t reflect.Type) Kind {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Map:
		return KindMap
	case reflect.Slice:
		return KindSequence
	case reflect.Bool:
		return KindBool
	case reflect.Int, reflect.Int32, reflect.Int64:
		return KindInt
	case reflect.Float32, reflect.Float64:
		return KindFloat
	default:
		return KindString
	}
}
