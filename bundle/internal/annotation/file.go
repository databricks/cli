package annotation

// TypeAnnotation holds the documentation for one Go type. Self documents the
// type itself — it is applied to the type's JSON-schema $defs entry and is
// where enum values live — and Fields documents each of the type's fields by
// JSON name.
type TypeAnnotation struct {
	Self   Descriptor            `json:"type,omitempty"`
	Fields map[string]Descriptor `json:"fields,omitempty"`
}

// File is the in-memory annotations, keyed by Go type path, e.g.
// "github.com/databricks/cli/bundle/config.Bundle".
type File map[string]TypeAnnotation

// SetField stores a descriptor for a field of typeKey, allocating the entry
// and its field map as needed.
func (f File) SetField(typeKey, name string, d Descriptor) {
	ta := f[typeKey]
	if ta.Fields == nil {
		ta.Fields = map[string]Descriptor{}
	}
	ta.Fields[name] = d
	f[typeKey] = ta
}

// SetSelf stores the descriptor for the type itself.
func (f File) SetSelf(typeKey string, d Descriptor) {
	ta := f[typeKey]
	ta.Self = d
	f[typeKey] = ta
}
