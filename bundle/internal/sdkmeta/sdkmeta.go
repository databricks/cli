// Package sdkmeta maps the API types used in the bundle configuration to the
// schemas of .codegen/cli.json and to the names they had in the previous Go SDK.
//
// Bundle configuration embeds sdk-go JSON mirrors (libs/sdktypes). Their names
// follow sdk-go (e.g. jobs.TaskSettings), while cli.json schemas and the
// published JSON schema / Python classes use the previous names (jobs.Task).
// The mapping is derived by walking the Go type graph of the bundle
// configuration in parallel with the cli.json schema graph: every field of a
// mapped type that refers to another schema maps the field's Go type to that
// schema. The walk starts from the explicit roots in roots.go.
package sdkmeta

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/structs/structtag"
)

// LegacyPkgPrefix is the package path prefix of the previous Go SDK. Schema
// definition names keep using it so that generated artifacts stay stable.
const LegacyPkgPrefix = "github.com/databricks/databricks-sdk-go/service/"

// Meta holds the mapping for one configuration root type.
type Meta struct {
	schemas map[string]*clijson.SchemaJSON
	names   map[reflect.Type]string
	// order records the walk order of types, see Order.
	order map[reflect.Type]int
	// alt lists, for types that sdk-go unified from several previous types,
	// the other schemas they stand for (see legacyFile.FieldRefs).
	alt map[reflect.Type][]string
}

// legacyFile is the format of legacy.json: the schema names API types had in
// the previous SDK, and their required fields. sdk-go and its JSON mirrors
// mark every field optional, and cli.json carries neither list element refs
// nor required fields, so this information is recorded here. New types not
// listed are mapped by following cli.json refs or fall back to their own name.
type legacyFile struct {
	// Names maps a Go type path to a cli.json schema name.
	Names map[string]string `json:"names"`
	// Required maps a cli.json schema name to its required fields.
	Required map[string][]string `json:"required"`
	// FieldRefs maps "<schema>.<field>" to the schema the field referred to
	// where sdk-go changed the field's type: an enum replaced by a plain
	// string, or two messages unified into one (jobs.TaskEmailNotifications
	// and jobs.JobEmailNotifications).
	FieldRefs map[string]string `json:"field_refs"`
}

//go:embed legacy.json
var legacyJSON []byte

var legacy = sync.OnceValue(func() legacyFile {
	var f legacyFile
	if err := json.Unmarshal(legacyJSON, &f); err != nil {
		panic(err)
	}
	return f
})

// Build walks root and maps every API type reachable from it.
func Build(root reflect.Type, schemas map[string]*clijson.SchemaJSON) (*Meta, error) {
	m := &Meta{schemas: schemas, names: map[reflect.Type]string{}, alt: map[reflect.Type][]string{}, order: map[reflect.Type]int{}}
	for t, name := range roots() {
		m.names[t] = name
	}
	lookup := func(t reflect.Type) string {
		if name, ok := m.names[t]; ok {
			return name
		}
		return legacy().Names[typeName(t)]
	}

	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		t = elem(t)
		if seen[t] {
			return
		}
		seen[t] = true
		m.order[t] = len(m.order)
		if isAPIType(t) {
			if name := lookup(t); name != "" {
				m.names[t] = name
			}
		}
		if t.Kind() != reflect.Struct || isNative(t) {
			return
		}
		for _, f := range fields(t) {
			ft := elem(f.field.Type)
			if !f.field.Anonymous && isAPIType(ft) && lookup(ft) == "" && f.owner != nil {
				if s := schemas[m.names[f.owner]]; s != nil {
					if fs := s.Fields[f.name]; fs != nil && fs.Ref != "" {
						m.names[ft] = fs.Ref
					}
				}
			}
			if !f.field.Anonymous && f.owner != nil {
				if ref, ok := FieldRef(m.names[f.owner], f.name); ok && lookup(ft) != ref && !slices.Contains(m.alt[ft], ref) {
					m.alt[ft] = append(m.alt[ft], ref)
				}
			}
			walk(ft)
		}
	}
	walk(root)
	return m, nil
}

// Name returns the cli.json schema name of t (e.g. "jobs.Task"), if t is a
// mapped API type.
func (m *Meta) Name(t reflect.Type) (string, bool) {
	name, ok := m.names[elem(t)]
	return name, ok
}

// Schema returns the cli.json schema of t, if t is a mapped API type.
func (m *Meta) Schema(t reflect.Type) (*clijson.SchemaJSON, bool) {
	name, ok := m.Name(t)
	if !ok {
		return nil, false
	}
	s, ok := m.schemas[name]
	return s, ok
}

// DefinitionName returns the type path the previous SDK type had, which is
// used as the JSON schema definition name (e.g.
// "github.com/databricks/databricks-sdk-go/service/jobs.Task").
func (m *Meta) DefinitionName(t reflect.Type) (string, bool) {
	name, ok := m.Name(t)
	if !ok {
		return "", false
	}
	if p, ok := nativeDefinitionNames[name]; ok {
		return p, true
	}
	return LegacyPkgPrefix + name, true
}

// Enum returns the enum values of t from cli.json.
func (m *Meta) Enum(t reflect.Type) ([]string, bool) {
	s, ok := m.Schema(t)
	if !ok || len(s.Enum) == 0 {
		return nil, false
	}
	return s.Enum, true
}

// SchemaHasField reports whether the named cli.json schema defines field. It
// reports true for names without a schema with fields (CLI types).
func (m *Meta) SchemaHasField(schema, field string) bool {
	s, ok := m.schemas[schema]
	if !ok || s.Fields == nil {
		return true
	}
	_, ok = s.Fields[field]
	return ok
}

// EnumOf returns the enum values of the named cli.json schema.
func (m *Meta) EnumOf(schema string) ([]string, bool) {
	s, ok := m.schemas[schema]
	if !ok || len(s.Enum) == 0 {
		return nil, false
	}
	return s.Enum, true
}

// Required returns the JSON names of the required fields of t, including
// those of API types embedded in t.
func (m *Meta) Required(t reflect.Type) []string {
	t = elem(t)
	var out []string
	queue := []reflect.Type{t}
	for len(queue) > 0 {
		ct := queue[0]
		queue = queue[1:]
		if name, ok := m.names[ct]; ok {
			out = append(out, legacy().Required[name]...)
		}
		if ct.Kind() != reflect.Struct {
			continue
		}
		for f := range ct.Fields() {
			if f.Anonymous {
				queue = append(queue, elem(f.Type))
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// FieldRef returns the schema the given field of schema referred to before
// sdk-go changed the field's type, if it did.
func FieldRef(schema, field string) (string, bool) {
	ref, ok := legacy().FieldRefs[schema+"."+field]
	return ref, ok
}

// Order returns the position of t in a depth-first walk of the root type in
// field declaration order, which is the order jsonschema.FromType visits
// types in. ok is false for types not reachable from the root.
func (m *Meta) Order(t reflect.Type) (int, bool) {
	i, ok := m.order[elem(t)]
	return i, ok
}

// RequiredFields returns the required fields of a cli.json schema.
func RequiredFields(schema string) []string {
	return legacy().Required[schema]
}

// AltSchemas returns the cli.json schemas of the previous types that t also
// stands for, beyond the one returned by Schema.
func (m *Meta) AltSchemas(t reflect.Type) []*clijson.SchemaJSON {
	var out []*clijson.SchemaJSON
	for _, name := range m.alt[elem(t)] {
		if s, ok := m.schemas[name]; ok {
			out = append(out, s)
		}
	}
	return out
}

// FieldOwners returns, for each serialized field of t, the API type declaring
// it, if any.
func (m *Meta) FieldOwners(t reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	for _, f := range fields(elem(t)) {
		if !f.field.Anonymous && f.owner != nil {
			out[f.name] = f.owner
		}
	}
	return out
}

// Types returns all mapped types.
func (m *Meta) Types() map[reflect.Type]string {
	return m.names
}

type fieldInfo struct {
	name  string
	field reflect.StructField
	// owner is the mapped API type declaring the field, nil for CLI types.
	owner reflect.Type
}

// fields lists the serialized fields of t with embedded structs flattened
// breadth first (as encoding/json and the schema generator do), remembering
// for each field the nearest API type that declares it. Embedded fields are
// listed too so the caller can validate their mapping.
func fields(t reflect.Type) []fieldInfo {
	type item struct {
		t     reflect.Type
		owner reflect.Type
	}
	var out []fieldInfo
	seen := map[string]bool{}
	queue := []item{{t, nil}}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		owner := it.owner
		if isAPIType(it.t) {
			owner = it.t
		}
		for f := range it.t.Fields() {
			if f.Anonymous {
				out = append(out, fieldInfo{field: f})
				queue = append(queue, item{elem(f.Type), owner})
				continue
			}
			name := structtag.JSONTag(f.Tag.Get("json")).Name()
			if name == "" || name == "-" || !f.IsExported() || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, fieldInfo{name: name, field: f, owner: owner})
		}
	}
	return out
}

func elem(t reflect.Type) reflect.Type {
	for {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map:
			t = t.Elem()
		default:
			return t
		}
	}
}

const (
	mirrorPkgPrefix = "github.com/databricks/cli/libs/sdktypes/"
	sdkPkgPrefix    = "github.com/databricks/sdk-go/"
	nativePkgPath   = "github.com/databricks/sdk-go/core/types"
)

// isAPIType reports whether t comes from the API (sdk-go mirrors, sdk-go
// enums aliased by the mirrors, hand-written API packages) or is a CLI-owned
// stand-in for an API type registered in roots.
func isAPIType(t reflect.Type) bool {
	if t.Name() == "" {
		return false
	}
	p := t.PkgPath()
	if _, ok := roots()[t]; ok {
		return true
	}
	return strings.HasPrefix(p, mirrorPkgPrefix) || strings.HasPrefix(p, sdkPkgPrefix) || p == "github.com/databricks/cli/libs/qualitymonitors"
}

func isNative(t reflect.Type) bool {
	return t.PkgPath() == nativePkgPath
}

func typeName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}
