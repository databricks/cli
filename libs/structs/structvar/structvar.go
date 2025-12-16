package structvar

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// StructVar is a container holding a typed struct and a map of unresolved references.
type StructVar struct {
	Value any `json:"value"`

	// Refs holds unresolved references. Key is serialized PathNode pointing inside a struct (e.g. "name")
	// and value is either pure or multiple references string: "${resources.foo.jobs.id}" or "${a} ${b}"
	Refs map[string]string `json:"vars,omitempty"`
}

// StructVarJSON is the serialized form of StructVar for persisting in plan files.
type StructVarJSON struct {
	Value json.RawMessage   `json:"value"`
	Refs  map[string]string `json:"vars,omitempty"`
}

var ErrNotFound = errors.New("reference not found")

// NewStructVar creates a StructVar from a typed value.
func NewStructVar(value any, refs map[string]string) *StructVar {
	return &StructVar{
		Value: value,
		Refs:  refs,
	}
}

// ToJSON converts StructVar to StructVarJSON for serialization.
func (sv *StructVar) ToJSON() (*StructVarJSON, error) {
	data, err := json.Marshal(sv.Value)
	if err != nil {
		return nil, err
	}
	return &StructVarJSON{
		Value: data,
		Refs:  sv.Refs,
	}, nil
}

// ToStructVar converts StructVarJSON to StructVar for working with typed values.
// typ must be a pointer type (e.g., *jobs.JobSettings).
func (svj *StructVarJSON) ToStructVar(typ reflect.Type) (*StructVar, error) {
	if typ.Kind() != reflect.Pointer {
		return nil, fmt.Errorf("expecting pointer, got %s", typ.Kind())
	}
	ptr := reflect.New(typ.Elem())
	if err := json.Unmarshal(svj.Value, ptr.Interface()); err != nil {
		return nil, err
	}
	return &StructVar{
		Value: ptr.Interface(),
		Refs:  svj.Refs,
	}, nil
}

// ResolveRef resolves the given reference by finding it in Refs values and replacing it.
// It searches through the Refs map to find values that contain the reference,
// performs string replacement, sets the resolved value, and removes fully resolved refs.
// Returns an error if the reference is not found or if setting the value fails.
func (sv *StructVar) ResolveRef(reference string, value any) error {
	var foundAny bool

	// Find all refs that contain this reference
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
			err = structaccess.Set(sv.Value, pathNode, value)
			if err != nil {
				return fmt.Errorf("cannot set (%T).%s to %T (%#v): %w", sv.Value, pathNode.String(), value, value, err)
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
			err = structaccess.Set(sv.Value, pathNode, newValue)
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

// Cache is a thread-safe cache for StructVar instances keyed by resource key.
type Cache struct {
	m sync.Map
}

// Store stores a StructVar in the cache.
func (c *Cache) Store(key string, sv *StructVar) {
	c.m.Store(key, sv)
}

// Load retrieves a StructVar from the cache.
func (c *Cache) Load(key string) (*StructVar, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*StructVar), true
}

// SyncToJSON updates the StructVarJSON.Value from the StructVar.Value.
// Call this after ResolveRef to persist changes to the plan entry.
func (sv *StructVar) SyncToJSON(svj *StructVarJSON) error {
	data, err := json.Marshal(sv.Value)
	if err != nil {
		return err
	}
	svj.Value = data
	svj.Refs = sv.Refs
	return nil
}
