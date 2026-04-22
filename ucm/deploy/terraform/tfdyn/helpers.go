package tfdyn

import "github.com/databricks/cli/libs/dyn"

// appendString adds key=fallback to pairs when vin does not already define
// the field. If vin sets the field, the caller's value wins and its
// location is preserved.
func appendString(pairs *[]dyn.Pair, vin dyn.Value, key, fallback string) {
	v := vin.Get(key)
	if s, ok := v.AsString(); ok && s != "" {
		*pairs = append(*pairs, dyn.Pair{
			Key:   dyn.NewValue(key, v.Locations()),
			Value: v,
		})
		return
	}
	*pairs = append(*pairs, dyn.Pair{
		Key:   dyn.V(key),
		Value: dyn.V(fallback),
	})
}

// appendStringIfSet appends key=vin[key] when the field is a non-empty
// string. Missing or empty fields are skipped so the Terraform JSON does
// not carry empty `"comment": ""` noise.
func appendStringIfSet(pairs *[]dyn.Pair, vin dyn.Value, key string) {
	v := vin.Get(key)
	s, ok := v.AsString()
	if !ok || s == "" {
		return
	}
	*pairs = append(*pairs, dyn.Pair{
		Key:   dyn.NewValue(key, v.Locations()),
		Value: v,
	})
}

// appendBoolIfSet emits key=vin[key] when vin[key] is a bool true. A false
// value (the zero) is skipped so the Terraform JSON stays clean.
func appendBoolIfSet(pairs *[]dyn.Pair, vin dyn.Value, key string) {
	v := vin.Get(key)
	b, ok := v.AsBool()
	if !ok || !b {
		return
	}
	*pairs = append(*pairs, dyn.Pair{
		Key:   dyn.NewValue(key, v.Locations()),
		Value: v,
	})
}

// mapFromValue returns v as a map-typed dyn.Value when it holds at least
// one key; otherwise the second return is false so callers can skip
// emitting an empty map.
func mapFromValue(v dyn.Value) (dyn.Value, bool) {
	m, ok := v.AsMap()
	if !ok || m.Len() == 0 {
		return dyn.InvalidValue, false
	}
	return v, true
}
