package dresources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// stateHashPrefix marks a state value that holds a content hash instead of the
// raw value. Since this is part of the on-disk state format, changing it is not
// backwards compatible.
const stateHashPrefix = "sha256_hashed_in_state:"

// hashStateValue returns a content-hash placeholder ("sha256_hashed_in_state:<hex>") over the JSON
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

	// json.Marshal sorts map keys, so the digest is stable across runs for equal values.
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling value for hashing: %w", err)
	}
	sum := sha256.Sum256(data)

	return stateHashPrefix + hex.EncodeToString(sum[:]), nil
}

// CompactState returns a copy of state with every field declared in cfg.HashedInState
// replaced by a content hash, so the state persists only the hash and not the full
// contents. It is applied both before persisting state and to every value entering the
// state diff, so stored and compared values share one form. The caller's value is never
// mutated (it is reused for the deploy API call, which needs the full contents).
//
// Returns state unchanged when no fields are declared or state is not a non-nil pointer.
func CompactState(cfg *ResourceLifecycleConfig, state any) (any, error) {
	if cfg == nil || len(cfg.HashedInState) == 0 {
		return state, nil
	}

	rv := reflect.ValueOf(state)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return state, nil
	}

	// Shallow copy so the caller's value (reused for the deploy) is untouched. This is
	// safe only because every hashed_in_state field is a top-level scalar (e.g.
	// serialized_dashboard): SetByString overwrites it on the copy directly. A field
	// reached through a shared pointer/slice/map would need a deep copy here.
	out := reflect.New(rv.Type().Elem())
	out.Elem().Set(rv.Elem())
	compacted := out.Interface()

	for _, field := range cfg.HashedInState {
		// The shallow copy above only isolates top-level fields: SetByString on a
		// depth-1 path reassigns the field on the copy, but a nested path (e.g.
		// "foo.bar") is reached through a pointer/slice/map still shared with the
		// caller's value, so hashing it would mutate the value reused for the deploy
		// API call. Reject nested paths loudly instead of corrupting state silently.
		path, err := structpath.ParsePath(field)
		if err != nil {
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
		if path.Len() != 1 {
			return nil, fmt.Errorf("hashed_in_state field %q must be a top-level field", field)
		}

		current, err := structaccess.GetByString(compacted, field)
		if err != nil {
			if _, ok := errors.AsType[*structaccess.NotFoundError](err); ok {
				continue
			}
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
		hashed, err := hashStateValue(current)
		if err != nil {
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
		if err := structaccess.SetByString(compacted, field, hashed); err != nil {
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
	}

	return compacted, nil
}
