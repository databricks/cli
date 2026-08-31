package testserver

import "encoding/json"

// mergeInto applies the fields a request body carries over an existing resource, leaving the
// rest as they were.
//
// It models what an update does to a field the body omits: nothing. A client sends its whole
// desired state, but the SDK request types are omitempty, so a field the config cleared is
// absent from the body and the backend keeps the value it had -- which is why clearing such a
// field never converges. A handler that unmarshals the body into a fresh struct instead
// applies the zero value, so the fake accepts a clear the real API ignores.
//
// Arrays are replaced whole, as the backend does: an element is not separately addressable,
// so a list is only ever sent entire.
func mergeInto[T any](existing T, body []byte) (T, error) {
	var out T
	stored, err := json.Marshal(existing)
	if err != nil {
		return out, err
	}

	var base, patch any
	if err := json.Unmarshal(stored, &base); err != nil {
		return out, err
	}
	if err := json.Unmarshal(body, &patch); err != nil {
		return out, err
	}

	merged, err := json.Marshal(mergeValue(base, patch))
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(merged, &out)
	return out, err
}

// mergeValue applies patch over base, recursing into objects and replacing anything else.
func mergeValue(base, patch any) any {
	baseMap, baseIsMap := base.(map[string]any)
	patchMap, patchIsMap := patch.(map[string]any)
	if !baseIsMap || !patchIsMap {
		return patch
	}
	for key, value := range patchMap {
		if existing, ok := baseMap[key]; ok {
			baseMap[key] = mergeValue(existing, value)
			continue
		}
		baseMap[key] = value
	}
	return baseMap
}
