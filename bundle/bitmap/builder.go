package bitmap

import (
	"slices"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// Builder computes the config bits of a bundle as the user wrote it. Mark runs
// on the loaded configuration, and on the resources Python adds, before any
// mutator fills in defaults. A field holding a variable reference has no value
// yet at that point, so Mark keeps it aside, and ResolveVariables marks the
// variable's value once it is known. Bits are only ever set, never cleared.
// Mark, MarkChanges and ResolveVariables do nothing on a nil Builder: loading
// creates it, and a bundle that was not loaded has no bitmap.
type Builder struct {
	index map[string]int
	bits  []bool

	// refs are the fields Mark found holding a pure ${var.*} reference.
	refs map[varRef]struct{}
}

// varRef is a field that held a pure ${var.*} reference.
type varRef struct {
	pattern   string // the field's schema pattern
	reference string // e.g. "${var.flag}"
}

// varPrefix and variablesPrefix start the two spellings of a variable
// reference: ${var.name} and ${variables.name.value} (or .default).
var (
	varPrefix       = dyn.NewPath(dyn.Key("var"))
	variablesPrefix = dyn.NewPath(dyn.Key("variables"))
)

// NewBuilder returns a Builder for schema with no bits set.
func NewBuilder(schema []string) *Builder {
	return &Builder{
		index: indexSchema(schema),
		bits:  make([]bool, len(schema)),
		refs:  map[varRef]struct{}{},
	}
}

// Mark sets the bits of the fields present in cfg, a configuration tree rooted
// at the configuration root. The config is walked as its dyn.Value rather than
// the typed struct so that presence is exactly what the config holds: an
// interface-typed field (variables.*.default) or an empty container ({} or [])
// counts, and a zero value counts only when it is written out.
func (b *Builder) Mark(cfg dyn.Value) {
	if b == nil {
		return
	}
	b.markChildren(cfg, "", false)
}

// ResolveVariables marks the values of the fields Mark found holding a
// variable reference (${var.*} or ${variables.*}), reading each variable's value from cfg, the processed
// configuration. The values are the ones the user supplied (default, --var,
// environment or file), whatever mutators later do to the field. A reference
// that does not resolve keeps only the presence bit Mark set.
func (b *Builder) ResolveVariables(cfg dyn.Value) {
	if b == nil {
		return
	}
	refs := b.refs
	b.refs = map[varRef]struct{}{}
	for ref := range refs {
		v, err := dynvar.Resolve(dyn.V(ref.reference), func(p dyn.Path) (dyn.Value, error) {
			return variableValue(cfg, p)
		})
		if err != nil {
			continue
		}
		// Variable values are not normalized against the fields they end up in:
		// a value from --var or the environment is a string, and so is a quoted
		// bool inside a complex variable. Coerce them where a bool field holds
		// them, the way the loader does.
		b.mark(v, ref.pattern, true)
	}
}

// coerceBool returns v as a bool when pattern is a bool field and v is a string
// the loader would read as one ("true", "false", ...), and v unchanged otherwise.
func coerceBool(v dyn.Value, index map[string]int, pattern string) dyn.Value {
	if _, isBool := index[pattern+trueSuffix]; !isBool || v.Kind() != dyn.KindString {
		return v
	}
	normalized, diags := convert.Normalize(false, v)
	if diags.HasError() {
		return v
	}
	return normalized
}

// variableValue looks up the value of a variable reference in cfg. cfg keeps
// ${var.name...} under variables.name.value; ${variables.name...} is a path into
// cfg as written.
func variableValue(cfg dyn.Value, p dyn.Path) (dyn.Value, error) {
	switch {
	case len(p) >= 2 && p.HasPrefix(varPrefix):
		path := append(dyn.NewPath(dyn.Key("variables"), p[1], dyn.Key("value")), p[2:]...)
		return dyn.GetByPath(cfg, path)
	case len(p) >= 3 && p.HasPrefix(variablesPrefix):
		return dyn.GetByPath(cfg, p)
	default:
		return dyn.InvalidValue, dynvar.ErrSkipResolution
	}
}

// Bits returns a bit per schema entry: the bits Mark and ResolveVariables set,
// together with those of the measured telemetry fields, namespaced under
// "telemetry.". A "=true"/"=false" entry is set when the bool held that value.
func (b *Builder) Bits(telemetry any) ([]bool, error) {
	bits := slices.Clone(b.bits)
	if err := markTelemetry(telemetry, b.index, bits); err != nil {
		return nil, err
	}
	return bits, nil
}

// markChildren marks every child of v whose schema pattern is known. A map key
// resolves to the struct field of that name when the schema has one and to the
// ".*" wildcard otherwise; a sequence element resolves to "[*]".
func (b *Builder) markChildren(v dyn.Value, pattern string, coerce bool) {
	if m, ok := v.AsMap(); ok {
		for _, pair := range m.Pairs() {
			b.mark(pair.Value, b.childPattern(pattern, pair.Key.MustString()), coerce)
		}
	}
	if s, ok := v.AsSequence(); ok {
		for _, elem := range s {
			b.mark(elem, pattern+listSuffix, coerce)
		}
	}
}

// childPattern returns the schema pattern of the map entry key under pattern:
// the field of that name, or the ".*" wildcard of a map.
func (b *Builder) childPattern(pattern, key string) string {
	// A field name never contains these, but a map key may (a variable named
	// "a.description", a key "enabled=true"); joined literally, such a key would
	// name another schema entry.
	if !strings.ContainsAny(key, ".=[]*") {
		if child := joinField(pattern, key); hasEntry(b.index, child) {
			return child
		}
	}
	return joinField(pattern, "*")
}

// mark sets the bit for a present (non-null) value and recurses into it. A
// pattern outside the schema is not descended into: that covers the pruned
// subtrees and the contents of interface-typed fields, whose own bit is set.
func (b *Builder) mark(v dyn.Value, pattern string, coerce bool) {
	i, ok := b.index[pattern]
	if !ok || v.Kind() == dyn.KindNil {
		return
	}
	b.bits[i] = true
	if coerce {
		v = coerceBool(v, b.index, pattern)
	}
	if value, ok := v.AsBool(); ok {
		setBit(b.index, b.bits, pattern+valueSuffix(value))
	}
	if s, ok := v.AsString(); ok && isVariableReference(s) {
		b.refs[varRef{pattern: pattern, reference: s}] = struct{}{}
	}
	b.markChildren(v, pattern, coerce)
}

// isVariableReference reports whether s is exactly one variable reference,
// ${var.*} or ${variables.*}.
func isVariableReference(s string) bool {
	p, ok := dynvar.PureReferenceToPath(s)
	return ok && (p.HasPrefix(varPrefix) || p.HasPrefix(variablesPrefix))
}
