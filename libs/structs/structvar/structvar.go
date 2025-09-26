package structvar

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// StructVar is a container holding a struct and a map of unresolved references inside this struct
type StructVar struct {
	Config any `json:"config"`

	// Refs holds unresolved references. Key is serialized PathNode pointing inside a struct (e.g. "name")
	// and value is either pure or multiple references string: "${resources.foo.jobs.id}" or "${a} ${b}"
	Refs map[string]string `json:"vars,omitempty"`
}

var ErrNotFound = errors.New("reference not found")

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
			err = structaccess.Set(sv.Config, pathNode, value)
			if err != nil {
				return fmt.Errorf("cannot set (%T).%s to %T (%#v): %w\nsv.Config = %#v", sv.Config, pathNode.String(), value, value, err, sv.Config)
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
			err = structaccess.Set(sv.Config, pathNode, newValue)
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
