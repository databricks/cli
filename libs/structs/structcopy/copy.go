// Package structcopy copies the fields two struct types have in common.
//
// The SDK describes one resource with several structs: a response, a create request, an
// update request. A field they share has the same Go name, JSON name and type in each, so
// a [Copier] matches fields by name and is told about the exceptions only. A field the SDK
// adds to both structs is copied without a code change.
package structcopy

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/structs/structtag"
)

const forceSendFieldsName = "ForceSendFields"

// Copier copies the fields that Src and Dst have in common.
type Copier[Src, Dst any] struct {
	plan func() (*plan, error)
}

type options struct {
	skip      []string
	forceSend []string
}

// Option names destination fields that a [Copier] treats differently.
type Option func(*options)

// Skip leaves the named destination fields unset although the source has them.
func Skip(fields ...string) Option {
	return func(o *options) {
		o.skip = append(o.skip, fields...)
	}
}

// ForceSend lists the named destination fields in ForceSendFields on every copy, so the
// SDK sends them even when they are zero.
func ForceSend(fields ...string) Option {
	return func(o *options) {
		o.forceSend = append(o.forceSend, fields...)
	}
}

// New returns a Copier from Src to Dst. The types are inspected on first use, so declaring
// a Copier in a package-level variable costs nothing at startup. An invalid Copier makes
// [Copier.Copy] panic: call [Copier.Describe] from a test to find out before it runs.
func New[Src, Dst any](opts ...Option) *Copier[Src, Dst] {
	return &Copier[Src, Dst]{
		plan: sync.OnceValues(func() (*plan, error) {
			var o options
			for _, opt := range opts {
				opt(&o)
			}
			src, dst := reflect.TypeFor[Src](), reflect.TypeFor[Dst]()
			p, err := newPlan(src, dst, o)
			if err != nil {
				return nil, fmt.Errorf("structcopy: %s -> %s: %w", src, dst, err)
			}
			return p, nil
		}),
	}
}

// Copy returns a new Dst holding the fields of src that Dst has as well. The copy is
// shallow: maps, slices and pointers are shared with src.
//
// ForceSendFields is not copied but derived: it lists the [ForceSend] fields, followed by
// the copied fields that src lists. The SDK marshaller rejects a name its struct lacks
// and would send the zero value of a field that was left unset.
func (c *Copier[Src, Dst]) Copy(src *Src) *Dst {
	p, err := c.plan()
	if err != nil {
		panic(err)
	}
	dst := new(Dst)
	p.apply(reflect.ValueOf(src).Elem(), reflect.ValueOf(dst).Elem())
	return dst
}

// Describe lists what the Copier does with every field of either struct, one line per
// field, or reports why the Copier is invalid.
func (c *Copier[Src, Dst]) Describe() (string, error) {
	p, err := c.plan()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s -> %s\n", p.src, p.dst)
	for _, f := range p.fields {
		fmt.Fprintf(&b, "  %s: %s\n", f.name, f.outcome)
	}
	return b.String(), nil
}

// plan is what a Copier does, worked out once from the two types.
type plan struct {
	src, dst reflect.Type

	// fields describes every field of either struct, sorted by name.
	fields []fieldOutcome
	steps  []copyStep
	forced []forceSendSlot
}

type fieldOutcome struct {
	name    string
	outcome string
}

type copyStep struct {
	src, dst []int

	// convertTo is set when the field types differ, as with the enum types the SDK
	// declares once per request and response.
	convertTo reflect.Type
}

// forceSendSlot is one ForceSendFields of the destination. There can be several, since a
// struct that embeds another keeps a ForceSendFields per struct.
type forceSendSlot struct {
	index  []int
	always []string
	from   []forceSendSource
}

// forceSendSource is a ForceSendFields of the source together with the fields it governs
// that were copied into the slot.
type forceSendSource struct {
	index  []int
	copied []string
}

func newPlan(src, dst reflect.Type, o options) (*plan, error) {
	srcFields, err := fieldsOf(src)
	if err != nil {
		return nil, err
	}
	dstFields, err := fieldsOf(dst)
	if err != nil {
		return nil, err
	}

	for _, name := range o.skip {
		if _, ok := dstFields[name]; !ok {
			return nil, fmt.Errorf("cannot skip %s: the destination has no such field", name)
		}
		if _, ok := srcFields[name]; !ok {
			return nil, fmt.Errorf("cannot skip %s: the source has no such field, so it is not copied anyway", name)
		}
	}

	p := &plan{src: src, dst: dst}

	for _, name := range o.forceSend {
		d, ok := dstFields[name]
		if !ok {
			return nil, fmt.Errorf("cannot force-send %s: the destination has no such field", name)
		}
		if !isBasic(d.typ.Kind()) {
			return nil, fmt.Errorf("cannot force-send %s: ForceSendFields has no effect on a %s", name, d.typ.Kind())
		}
		if d.forceSend == nil {
			return nil, fmt.Errorf("cannot force-send %s: the struct that declares it has no ForceSendFields", name)
		}
		slot := p.slot(d.forceSend)
		slot.always = append(slot.always, name)
	}

	names := slices.Collect(maps.Keys(dstFields))
	for name := range srcFields {
		if _, ok := dstFields[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	for _, name := range names {
		d, inDestination := dstFields[name]
		s, inSource := srcFields[name]

		var outcome string
		switch {
		case !inDestination:
			outcome = "not in destination"
		case slices.Contains(o.skip, name):
			outcome = "skipped"
		case !inSource:
			outcome = "not in source"
		default:
			step, err := newCopyStep(s, d)
			if err != nil {
				return nil, err
			}
			p.steps = append(p.steps, step)
			outcome = "copied"
			if step.convertTo != nil {
				outcome += ", converted from " + s.typ.String()
			}
			if s.forceSend != nil && d.forceSend != nil {
				p.slot(d.forceSend).copiedFrom(s.forceSend, name)
			}
		}
		if slices.Contains(o.forceSend, name) {
			outcome += ", always sent"
		}
		p.fields = append(p.fields, fieldOutcome{name: name, outcome: outcome})
	}

	return p, nil
}

func newCopyStep(s, d field) (copyStep, error) {
	step := copyStep{src: s.index, dst: d.index}
	if d.jsonName == "-" {
		// A path or query parameter addresses the resource, so its value is not the
		// caller's field of the same name: UpdateCatalog.Name is the catalog to update.
		return step, fmt.Errorf("%s is not in the destination's JSON (json:\"-\"); skip it and set it by hand", d.name)
	}
	if s.jsonName != d.jsonName {
		return step, fmt.Errorf("%s is %q in the source's JSON but %q in the destination's; skip it and set it by hand", d.name, s.jsonName, d.jsonName)
	}
	if s.typ == d.typ {
		return step, nil
	}
	if s.typ.Kind() != d.typ.Kind() || !isBasic(d.typ.Kind()) {
		return step, fmt.Errorf("%s is %s in the source but %s in the destination; skip it and set it by hand", d.name, s.typ, d.typ)
	}
	step.convertTo = d.typ
	return step, nil
}

// slot returns the forceSendSlot for the destination's ForceSendFields at index.
func (p *plan) slot(index []int) *forceSendSlot {
	for i := range p.forced {
		if slices.Equal(p.forced[i].index, index) {
			return &p.forced[i]
		}
	}
	p.forced = append(p.forced, forceSendSlot{index: index})
	return &p.forced[len(p.forced)-1]
}

// copiedFrom records that the named field is copied into the slot from a struct whose
// ForceSendFields is at index.
func (s *forceSendSlot) copiedFrom(index []int, name string) {
	for i := range s.from {
		if slices.Equal(s.from[i].index, index) {
			s.from[i].copied = append(s.from[i].copied, name)
			return
		}
	}
	s.from = append(s.from, forceSendSource{index: index, copied: []string{name}})
}

func (p *plan) apply(src, dst reflect.Value) {
	for _, step := range p.steps {
		v := src.FieldByIndex(step.src)
		if step.convertTo != nil {
			v = v.Convert(step.convertTo)
		}
		dst.FieldByIndex(step.dst).Set(v)
	}

	for _, slot := range p.forced {
		names := slices.Clone(slot.always)
		for _, from := range slot.from {
			listed, _ := reflect.TypeAssert[[]string](src.FieldByIndex(from.index))
			for _, name := range listed {
				if slices.Contains(from.copied, name) && !slices.Contains(names, name) {
					names = append(names, name)
				}
			}
		}
		if len(names) > 0 {
			dst.FieldByIndex(slot.index).Set(reflect.ValueOf(names))
		}
	}
}

// field is a field that a composite literal of its struct could set by name: declared by
// the struct itself or promoted from a struct it embeds.
type field struct {
	name     string
	index    []int
	typ      reflect.Type
	jsonName string

	// forceSend is the index of the ForceSendFields that governs the field: the one
	// declared by the same struct. It is nil if that struct declares none.
	forceSend []int
}

func fieldsOf(t reflect.Type) (map[string]field, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s is not a struct", t)
	}

	fields := map[string]field{}
	for _, sf := range reflect.VisibleFields(t) {
		if sf.Anonymous {
			if sf.Type.Kind() == reflect.Struct {
				// Its fields are promoted, so they are visited on their own.
				continue
			}
			if sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct {
				return nil, fmt.Errorf("%s embeds %s; embedded pointers are not supported", t, sf.Type)
			}
		}
		if !sf.IsExported() || sf.Name == forceSendFieldsName {
			continue
		}

		parentIndex := sf.Index[:len(sf.Index)-1]
		parent := t
		if len(parentIndex) > 0 {
			parent = t.FieldByIndex(parentIndex).Type
		}
		var forceSend []int
		if fsf, ok := parent.FieldByName(forceSendFieldsName); ok && len(fsf.Index) == 1 && fsf.Type == reflect.TypeFor[[]string]() {
			forceSend = append(slices.Clone(parentIndex), fsf.Index[0])
		}

		fields[sf.Name] = field{
			name:      sf.Name,
			index:     sf.Index,
			typ:       sf.Type,
			jsonName:  structtag.JSONTag(sf.Tag.Get("json")).Name(),
			forceSend: forceSend,
		}
	}
	return fields, nil
}

// isBasic reports whether ForceSendFields applies to a field of this kind; see isBasicType
// in the SDK's marshal package.
func isBasic(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}
