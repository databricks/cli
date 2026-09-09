package dresources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// stateHashPrefix marks a state value that holds a content hash instead of the
// raw value. Since this is part of the on-disk state format, changing it is not
// backwards compatible.
const stateHashPrefix = "sha256:"

// stateHashPlaceholderLen is the exact size of a placeholder: the prefix followed by a
// SHA-256 digest in hex (sha256.Size bytes, two characters each).
const stateHashPlaceholderLen = len(stateHashPrefix) + sha256.Size*2

// isStateHashPlaceholder reports whether s matches ^sha256:[a-f0-9]{64}$ exactly, so only
// values this package produced count as already-hashed, not content that shares the prefix.
func isStateHashPlaceholder(s string) bool {
	if len(s) != stateHashPlaceholderLen || !strings.HasPrefix(s, stateHashPrefix) {
		return false
	}
	for i := len(stateHashPrefix); i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// hashStateValue returns a content-hash placeholder ("sha256:<hex>") for s, used to store
// large, equality-only string fields (e.g. a dashboard's serialized_dashboard) compactly in
// state instead of their full contents. hashed_fields values are serialized to a string
// before the deploy engine runs, so this operates on a plain string.
//
// It is idempotent and stable: an empty string, a value already a placeholder, and a value
// no larger than a placeholder are returned unchanged.
func hashStateValue(s string) string {
	if s == "" || isStateHashPlaceholder(s) || len(s) <= stateHashPlaceholderLen {
		return s
	}
	sum := sha256.Sum256([]byte(s))
	return stateHashPrefix + hex.EncodeToString(sum[:])
}

// CompactState returns a copy of state with every field declared in cfg.HashedFields
// replaced by a content hash, so the state persists only the hash and not the full
// contents. A field whose contents are already no larger than a placeholder is left as
// is (see hashStateValue). It is applied both before persisting state and to every value
// entering the state diff, so stored and compared values share one form. The caller's
// value is never mutated (it is reused for the deploy API call, which needs the full
// contents).
//
// Returns state unchanged when no fields are declared or state is not a non-nil pointer.
func CompactState(cfg *ResourceLifecycleConfig, state any) (any, error) {
	if cfg == nil || len(cfg.HashedFields) == 0 {
		return state, nil
	}

	rv := reflect.ValueOf(state)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return state, nil
	}

	// Shallow copy so the caller's value (reused for the deploy) is untouched. This is
	// safe only because every hashed_fields field is a top-level field (e.g.
	// serialized_dashboard): SetByString overwrites it on the copy directly. A field
	// reached through a shared pointer/slice/map would need a deep copy here.
	out := reflect.New(rv.Type().Elem())
	out.Elem().Set(rv.Elem())
	compacted := out.Interface()

	for _, field := range cfg.HashedFields {
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
			return nil, fmt.Errorf("hashed_fields field %q must be a top-level field", field)
		}

		current, err := structaccess.GetByString(compacted, field)
		if err != nil {
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
		if current == nil {
			// Field is unset; nothing to hash.
			continue
		}
		s, ok := current.(string)
		if !ok {
			// hashed_fields values are serialized to a string before the deploy engine
			// runs (e.g. ConfigureDashboardSerializedDashboard), so a non-string here is a
			// broken invariant / programming error, not a user error — fail loudly.
			panic(fmt.Sprintf("hashed_fields field %q must be a string, got %T", field, current))
		}
		if err := structaccess.SetByString(compacted, field, hashStateValue(s)); err != nil {
			return nil, fmt.Errorf("compacting state field %q: %w", field, err)
		}
	}

	return compacted, nil
}
