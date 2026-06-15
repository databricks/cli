package dresources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// stateHashPrefix marks a state value that holds a content hash instead of the
// full content. It is part of the on-disk state format, so changing it is a
// backward-incompatible change.
const stateHashPrefix = "sha256:"

// hashStateValue returns a content-hash placeholder ("sha256:<hex>") over the JSON
// encoding of v. It is used to store large, equality-only fields (e.g. a dashboard's
// serialized_dashboard) compactly in state instead of their full contents.
//
// It is idempotent and stable: nil, an empty string, and a value that is already a
// placeholder are returned unchanged, so re-compacting an already-compact state and
// comparing a fresh config against stored state both behave predictably.
func hashStateValue(v any) (any, error) {
	if s, ok := v.(string); ok {
		if s == "" || strings.HasPrefix(s, stateHashPrefix) {
			return v, nil
		}
	}
	if v == nil {
		return v, nil
	}

	// json.Marshal is deterministic (map keys are sorted), so equal content always
	// produces an equal hash across runs and platforms.
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return stateHashPrefix + hex.EncodeToString(sum[:]), nil
}
