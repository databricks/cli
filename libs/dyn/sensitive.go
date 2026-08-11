package dyn

import "slices"

// SensitiveValueRedacted is the placeholder emitted in JSON/YAML output
// whenever a sensitive string value is serialized.
const SensitiveValueRedacted = "<redacted value>"

// secretString is the internal storage type for sensitive string values.
// Using a distinct Go type (rather than a plain string plus a flag) makes
// sensitivity impossible to strip by accident: every path that copies v.v
// preserves the wrapper automatically, and a switch on v.v that handles
// `string` but not `secretString` will miss it — making omissions auditable.
//
// Kind() still returns KindString so all existing switch-on-Kind logic
// continues to work unchanged.
type secretString struct {
	value string
}

// NewSensitiveValue returns a KindString Value whose content is treated as
// sensitive. JSON and YAML serializers replace it with [SensitiveValueRedacted];
// AsString / MustString still return the real value for use in the deployment
// pipeline.
func NewSensitiveValue(s string, loc []Location) Value {
	return Value{
		v: secretString{s},
		k: KindString,
		l: slices.Clone(loc),
	}
}
