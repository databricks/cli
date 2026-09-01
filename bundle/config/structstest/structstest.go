// Package structstest checks that the libs/structs packages agree with encoding/json
// about a type.
//
// encoding/json is the oracle: it decides which fields exist on the wire, under which
// names, and at which paths. Every libs/structs package claims to speak that same
// vocabulary -- structwalk enumerates it, structaccess reads and writes it, structdiff
// reports changes in it -- so any disagreement is a bug in one of them, or in the type.
//
// The package is consumed by tests (bundle/config and bundle/direct/dresources feed it
// the bundle's own resource, state and remote types) but is not itself a _test package,
// because those two callers live in different trees.
package structstest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// Report lists the disagreements found for one type. A field is identified by the JSON
// path encoding/json puts it at, which is the only name all the packages share.
type Report struct {
	// WalkMissing are paths encoding/json emits that structwalk never visits, so
	// structdiff cannot see a change to them either.
	WalkMissing []string

	// WalkExtra are paths structwalk visits that encoding/json does not emit. Usually a
	// type that embeds another with its own MarshalJSON and does not define one itself:
	// the embedded marshaler takes over and the outer fields never reach the wire.
	WalkExtra []string

	// GetFailed are paths encoding/json emits that structaccess.Get cannot resolve.
	GetFailed []string

	// ValidateFailed are paths encoding/json emits that structaccess.ValidatePath
	// rejects against the type, even though Get resolves them on the value.
	ValidateFailed []string

	// ValueMismatch are paths where structaccess.Get and encoding/json disagree about
	// the value stored at the path.
	ValueMismatch []string

	// InsideFreeFormField are paths that sit inside a free-form any field. structwalk does not
	// traverse an interface and structaccess cannot validate a path through one, so nothing
	// below such a field is visible to the packages -- a serialized dashboard or a cluster
	// policy definition authored as inline YAML is opaque to them. One known limitation rather
	// than a list of paths, which would depend on the filler's choice of key.
	InsideFreeFormField []string

	// SelfMarshalingScalars are paths whose Go type is a struct that marshals itself as a
	// scalar through its own MarshalJSON -- duration.Duration and the SDK's time wrapper.
	// structwalk looks for scalar *fields* and finds none inside them, so it never visits
	// them and structdiff never reports drift on them. One known limitation rather than a
	// per-field list, because every new timestamp field in the SDK joins it.
	SelfMarshalingScalars []string
}

// Empty reports whether the type and the libs/structs packages agree completely.
func (r Report) Empty() bool {
	return len(r.WalkMissing) == 0 && len(r.WalkExtra) == 0 && len(r.GetFailed) == 0 &&
		len(r.ValidateFailed) == 0 && len(r.ValueMismatch) == 0 && len(r.SelfMarshalingScalars) == 0 &&
		len(r.InsideFreeFormField) == 0
}

// String renders the report as one indented line per category, for a test failure message.
func (r Report) String() string {
	var b strings.Builder
	for _, s := range []struct {
		label string
		paths []string
	}{
		{"structwalk does not visit, encoding/json emits", r.WalkMissing},
		{"structwalk visits, encoding/json does not emit", r.WalkExtra},
		{"structaccess.Get cannot resolve", r.GetFailed},
		{"structaccess.ValidatePath rejects", r.ValidateFailed},
		{"structaccess.Get and encoding/json disagree on the value at", r.ValueMismatch},
		{"marshals itself as a scalar, so structwalk never visits", r.SelfMarshalingScalars},
		{"sits inside a free-form any field, which the packages do not look into", r.InsideFreeFormField},
	} {
		if len(s.paths) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n  %s (%d):", s.label, len(s.paths))
		for _, p := range s.paths {
			fmt.Fprintf(&b, "\n    %s", p)
		}
	}
	return b.String()
}

// Check fills every field of a fresh value of type t with a non-zero value, marshals it
// with encoding/json, and compares the result against what structwalk and structaccess
// make of the same value. t must be a struct or a pointer to one.
func Check(t reflect.Type) (Report, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Report{}, fmt.Errorf("structstest: %s is not a struct", t)
	}

	ptr := reflect.New(t)
	FillNonZero(ptr.Elem())
	v := ptr.Interface()

	jsonLeaves, marks, err := jsonLeaves(v)
	if err != nil {
		return Report{}, err
	}

	walkLeaves := map[string]string{}
	err = structwalk.Walk(v, func(path *structpath.PathNode, val any, _ *reflect.StructField) {
		walkLeaves[path.String()] = render(val)
	})
	if err != nil {
		return Report{}, fmt.Errorf("structstest: walk %s: %w", t, err)
	}

	var report Report
	for path, want := range jsonLeaves {
		if marks.freeForm[path] {
			report.InsideFreeFormField = append(report.InsideFreeFormField, path)
			continue
		}
		if marks.selfMarshaling[path] {
			report.SelfMarshalingScalars = append(report.SelfMarshalingScalars, path)
			continue
		}
		if _, ok := walkLeaves[path]; !ok {
			report.WalkMissing = append(report.WalkMissing, path)
		}

		node, err := structpath.ParsePath(path)
		if err != nil {
			report.GetFailed = append(report.GetFailed, path+": "+err.Error())
			continue
		}
		if skippedByTag(reflect.TypeOf(v), node) {
			// structaccess refuses bundle:"internal" and bundle:"readonly" fields by
			// design; encoding/json still emits them.
			continue
		}
		if err := structaccess.ValidatePath(reflect.TypeOf(v), node); err != nil {
			report.ValidateFailed = append(report.ValidateFailed, path+": "+err.Error())
		}
		got, err := structaccess.Get(v, node)
		if err != nil {
			report.GetFailed = append(report.GetFailed, path+": "+err.Error())
			continue
		}
		if render(got) != want {
			report.ValueMismatch = append(report.ValueMismatch,
				fmt.Sprintf("%s: structaccess=%s encoding/json=%s", path, render(got), want))
		}
	}
	for path := range walkLeaves {
		if _, ok := jsonLeaves[path]; !ok {
			if marks.freeFormField[path] {
				// The any field itself: the walk offers it as a scalar leaf while the wire format
				// carries whatever it holds, which is the same limitation seen from the other side.
				report.InsideFreeFormField = append(report.InsideFreeFormField, path)
				continue
			}
			report.WalkExtra = append(report.WalkExtra, path)
		}
	}

	slices.Sort(report.WalkMissing)
	slices.Sort(report.WalkExtra)
	slices.Sort(report.GetFailed)
	slices.Sort(report.ValidateFailed)
	slices.Sort(report.ValueMismatch)
	slices.Sort(report.SelfMarshalingScalars)
	return report, nil
}

// jsonLeaves marshals v and returns its scalar leaves keyed by the structpath rendering of
// their location, which is the dialect every libs/structs package speaks, plus the subset of
// those leaves whose Go type marshalled itself as a scalar.
// leafMarks records leaves that need a category of their own rather than a path-by-path
// comparison.
type leafMarks struct {
	// selfMarshaling are leaves whose Go type is a struct that marshalled itself as a scalar.
	selfMarshaling map[string]bool
	// freeForm are leaves below an any field; freeFormField holds the any fields themselves.
	freeForm      map[string]bool
	freeFormField map[string]bool
}

func jsonLeaves(v any) (map[string]string, leafMarks, error) {
	marks := leafMarks{
		selfMarshaling: map[string]bool{},
		freeForm:       map[string]bool{},
		freeFormField:  map[string]bool{},
	}

	blob, err := json.Marshal(v)
	if err != nil {
		return nil, marks, fmt.Errorf("structstest: marshal %T: %w", v, err)
	}
	var generic any
	if err := json.Unmarshal(blob, &generic); err != nil {
		return nil, marks, fmt.Errorf("structstest: unmarshal %T: %w", v, err)
	}
	out := map[string]string{}
	flatten(nil, reflect.TypeOf(v), generic, out, marks, false)
	return out, marks, nil
}

// flatten walks the decoded JSON alongside the Go type, because the path syntax for an
// object member depends on which one it is: a struct field is .name, a map entry is
// ['name'], and only the type knows the difference.
func flatten(path *structpath.PathNode, typ reflect.Type, v any, out map[string]string, marks leafMarks, freeForm bool) {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ != nil && typ.Kind() == reflect.Interface {
		// Everything below an any field is opaque to the packages.
		marks.freeFormField[path.String()] = true
		freeForm = true
	}

	switch value := v.(type) {
	case map[string]any:
		isMap := typ != nil && typ.Kind() == reflect.Map
		for key, member := range value {
			var next *structpath.PathNode
			var memberType reflect.Type
			if isMap {
				next = structpath.NewBracketString(path, key)
				memberType = typ.Elem()
			} else {
				next = structpath.NewStringKey(path, key)
				if typ != nil && typ.Kind() == reflect.Struct {
					if sf, ok := embeddedSliceField(typ, key); ok {
						// An EmbeddedSlice field is transparent by design: the walkers put
						// its elements at the parent path, while the wire format keeps the
						// __embed__ key. Follow the walkers, or every such type reads as a
						// disagreement when it is really the convention working.
						flatten(path, sf.Type, member, out, marks, freeForm)
						continue
					}
					if sf, _, ok := structaccess.FindStructFieldByKeyType(typ, key); ok {
						memberType = sf.Type
					}
				}
			}
			flatten(next, memberType, member, out, marks, freeForm)
		}
	case []any:
		var elemType reflect.Type
		if typ != nil && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) {
			elemType = typ.Elem()
		}
		for i, member := range value {
			flatten(structpath.NewIndex(path, i), elemType, member, out, marks, freeForm)
		}
	case nil:
		// A JSON null carries no scalar leaf.
	default:
		out[path.String()] = render(value)
		if freeForm {
			marks.freeForm[path.String()] = true
		}
		// A scalar on the wire whose Go type is a struct marshalled itself: the walkers
		// cannot see inside it.
		if typ != nil && typ.Kind() == reflect.Struct {
			marks.selfMarshaling[path.String()] = true
		}
	}
}

// embeddedSliceField reports whether key names the struct's EmbeddedSlice field.
func embeddedSliceField(typ reflect.Type, key string) (reflect.StructField, bool) {
	for sf := range typ.Fields() {
		if sf.Name != structaccess.EmbeddedSliceFieldName {
			continue
		}
		if structtag.JSONTag(sf.Tag.Get("json")).Name() == key {
			return sf, true
		}
	}
	return reflect.StructField{}, false
}

// render normalises a scalar so a value decoded from JSON and the same value read out of
// the struct compare equal: JSON numbers decode to float64, the struct holds int64 and
// friends, and a nil pointer reads back as nil.
func render(v any) string {
	if v == nil {
		return "<nil>"
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "<nil>"
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatFloat(float64(rv.Int()), 'g', -1, 64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatFloat(float64(rv.Uint()), 'g', -1, 64)
	default:
		return fmt.Sprintf("%v", rv.Interface())
	}
}

// skippedByTag reports whether the last segment of path names a field structaccess
// refuses on purpose. The lookup goes through embedded structs, so a field promoted from
// BaseResource is recognised too.
func skippedByTag(typ reflect.Type, path *structpath.PathNode) bool {
	nodes := path.AsSlice()
	cur := typ
	for i, node := range nodes {
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}
		key, ok := node.StringKey()
		if !ok || cur.Kind() != reflect.Struct {
			return false
		}
		sf, _, found := structaccess.FindStructFieldByKeyType(cur, key)
		if !found {
			// structaccess drops internal and readonly fields from the type-level
			// lookup as well, so a miss on the last segment is the tag talking.
			return i == len(nodes)-1 && taggedInternal(cur, key)
		}
		cur = sf.Type
	}
	return false
}

// taggedInternal reports whether the struct, or a struct it embeds, declares key with
// bundle:"internal" or bundle:"readonly".
func taggedInternal(typ reflect.Type, key string) bool {
	for sf := range typ.Fields() {
		if !sf.IsExported() {
			continue
		}
		if structtag.JSONTag(sf.Tag.Get("json")).Name() == key {
			bt := structtag.BundleTag(sf.Tag.Get("bundle"))
			if bt.Internal() || bt.ReadOnly() {
				return true
			}
		}
		if !sf.Anonymous {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && taggedInternal(ft, key) {
			return true
		}
	}
	return false
}

// FillNonZero populates every serializable field of v with a non-zero value, so that a
// field the packages disagree about is always observable rather than indistinguishable
// from an omitted zero. Recursion is bounded because SDK types are self-referential.
func FillNonZero(v reflect.Value) { fillNonZero(v, 0) }

func fillNonZero(v reflect.Value, depth int) {
	if depth > 5 {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.String:
		v.SetString("x")
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillNonZero(v.Elem(), depth+1)
	case reflect.Slice:
		elem := reflect.New(v.Type().Elem()).Elem()
		fillNonZero(elem, depth+1)
		v.Set(reflect.Append(v, elem))
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		val := reflect.New(v.Type().Elem()).Elem()
		fillNonZero(val, depth+1)
		v.SetMapIndex(reflect.ValueOf("k").Convert(v.Type().Key()), val)
	case reflect.Interface:
		// A free-form any field holds a composite in practice -- a cluster policy definition or
		// a serialized dashboard authored as inline YAML -- and that is the case worth covering,
		// because structwalk does not traverse into an interface and so cannot see any of it.
		v.Set(reflect.ValueOf(map[string]any{"k": "v"}))
	case reflect.Struct:
		for i := range v.Type().NumField() {
			sf := v.Type().Field(i)
			if !sf.IsExported() || sf.Name == "ForceSendFields" {
				continue
			}
			if structtag.JSONTag(sf.Tag.Get("json")).Name() == "-" {
				continue
			}
			fillNonZero(v.Field(i), depth+1)
		}
	default:
		// Kinds that do not appear in bundle or SDK types (chan, func, complex) stay zero.
	}
}

// Filter removes the known divergences from a report and returns the entries that matched
// nothing, so a caller can fail when a recorded divergence has been fixed and the entry is
// stale. Without that, a known-divergence list is an exemption rather than a ratchet.
//
// Prefix coverage applies only to the two walk categories, where a field lost wholesale takes
// all of its leaves with it. The categories that name a specific failure -- Get, ValidatePath,
// a value mismatch -- match exactly, so an entry cannot quietly absorb an unrelated failure at
// a path beneath it.
func (r Report) Filter(known []string) (Report, []string) {
	used := map[string]bool{}

	dropPrefix := func(paths []string) []string {
		var out []string
		for _, p := range paths {
			if match, ok := coveredBy(known, pathOf(p)); ok {
				used[match] = true
				continue
			}
			out = append(out, p)
		}
		return out
	}
	dropExact := func(paths []string) []string {
		var out []string
		for _, p := range paths {
			if slices.Contains(known, pathOf(p)) {
				used[pathOf(p)] = true
				continue
			}
			out = append(out, p)
		}
		return out
	}

	filtered := Report{
		WalkMissing:    dropPrefix(r.WalkMissing),
		WalkExtra:      dropPrefix(r.WalkExtra),
		GetFailed:      dropExact(r.GetFailed),
		ValidateFailed: dropExact(r.ValidateFailed),
		ValueMismatch:  dropExact(r.ValueMismatch),
		// These two are categories, not per-path lists, so the caller decides what to do with
		// them. Dropping them here would make that decision unreachable.
		SelfMarshalingScalars: r.SelfMarshalingScalars,
		InsideFreeFormField:   r.InsideFreeFormField,
	}

	var stale []string
	for _, k := range known {
		if !used[k] {
			stale = append(stale, k)
		}
	}
	return filtered, stale
}

// pathOf strips the explanatory suffix some categories append after a colon.
func pathOf(reported string) string {
	return strings.SplitN(reported, ":", 2)[0]
}

// coveredBy reports which entry covers path: the one that equals it, or the longest one that
// names a field it sits underneath. Longest wins so that overlapping entries -- "a" and "a.b"
// -- are each credited with what they alone cover, rather than the first one absorbing
// everything and leaving the other looking stale.
func coveredBy(known []string, path string) (string, bool) {
	best := ""
	for _, k := range known {
		if path == k {
			return k, true
		}
		if !strings.HasPrefix(path, k) || !strings.ContainsAny(path[len(k):len(k)+1], ".[") {
			continue
		}
		if len(k) > len(best) {
			best = k
		}
	}
	return best, best != ""
}
