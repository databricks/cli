package structvar

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// StructVar is a container holding a struct and a map of unresolved references inside this struct
type StructVar struct {
	Value json.RawMessage `json:"value"`

	// Refs holds unresolved references. Key is serialized PathNode pointing inside a struct (e.g. "name")
	// and value is either pure or multiple references string: "${resources.foo.jobs.id}" or "${a} ${b}"
	Refs map[string]string `json:"vars,omitempty"`

	valueCache any
}

var ErrNotFound = errors.New("reference not found")

// NewStructVar creates a StructVar from a typed value.
// Marshals value to JSON for serialization, keeps typed value in cache.
func NewStructVar(value any, refs map[string]string) (*StructVar, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &StructVar{
		Value:      data,
		Refs:       refs,
		valueCache: value,
	}, nil
}

// Load parses Value (json.RawMessage) into valueCache as the given type.
// Must be called before ResolveRef when reading from persisted state.
// typ must be a pointer type (e.g., *jobs.JobSettings).
func (sv *StructVar) Load(typ reflect.Type) error {
	if sv.valueCache != nil {
		return nil // Already loaded
	}
	if typ.Kind() != reflect.Pointer {
		return fmt.Errorf("expecting pointer, got %s", typ.Kind())
	}
	ptr := reflect.New(typ.Elem())
	if err := json.Unmarshal(sv.Value, ptr.Interface()); err != nil {
		return err
	}
	sv.valueCache = ptr.Interface()
	return nil
}

// Save marshals valueCache back to Value (json.RawMessage).
// Call after ResolveRef to persist changes.
func (sv *StructVar) Save() error {
	if sv.valueCache == nil {
		return nil
	}
	data, err := json.Marshal(sv.valueCache)
	if err != nil {
		return err
	}
	sv.Value = data
	return nil
}

// GetValue returns the cached typed value.
// Returns nil if Load hasn't been called and NewStructVar wasn't used.
func (sv *StructVar) GetValue() any {
	return sv.valueCache
}

// ResolveRef resolves the given reference by finding it in Refs values and replacing it.
// It searches through the Refs map to find values that contain the reference,
// performs string replacement, sets the resolved value, and removes fully resolved refs.
// Returns an error if the reference is not found or if setting the value fails.
func (sv *StructVar) ResolveRef(reference string, value any) error {
	var foundAny bool

	// Find all refs that contain this reference
	// QQQ we can add reverse index
	for pathKey, refValue := range sv.Refs {
		if !strings.Contains(refValue, reference) {
			continue
		}

		foundAny = true

		// Parse the path
		pathNode, err := structpath.Parse(pathKey)
		if err != nil {
			return fmt.Errorf("invalid path %q: %w", pathKey, err)
		}

		// Check if this is a pure reference (reference equals the entire value)
		if refValue == reference {
			// Pure reference - use original typed value
			err = structaccess.Set(sv.valueCache, pathNode, value)
			if err != nil {
				return fmt.Errorf("cannot set (%T).%s to %T (%#v): %w", sv.valueCache, pathNode.String(), value, value, err)
			}
			// Remove the fully resolved reference
			delete(sv.Refs, pathKey)
		} else {
			// Partial reference or multiple references - do string replacement
			valueStr, err := structaccess.ConvertToString(value)
			if err != nil {
				return fmt.Errorf("cannot set %s to %T (%#v): %w", pathNode.String(), value, value, err)
			}
			newValue := strings.ReplaceAll(refValue, reference, valueStr)

			// Set the updated string value
			err = structaccess.Set(sv.valueCache, pathNode, newValue)
			if err != nil {
				return fmt.Errorf("cannot update %s to string: %w", pathNode.String(), err)
			}

			// Check if fully resolved (no more ${} patterns)
			if !dynvar.ContainsVariableReference(newValue) {
				delete(sv.Refs, pathKey)
			} else {
				sv.Refs[pathKey] = newValue
			}
		}
	}

	if !foundAny {
		return ErrNotFound
	}

	return nil
}
