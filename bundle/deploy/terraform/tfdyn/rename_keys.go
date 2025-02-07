package tfdyn

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

// renameKeys renames keys in the given map value.
//
// Terraform resources sometimes use singular names for repeating blocks where the API
// definition uses the plural name. This function can convert between the two.
func renameKeys(v dyn.Value, rename map[string]string) (dyn.Value, error) {
	var err error
	acc := dyn.V(map[string]dyn.Value{})

	nv, err := dyn.Walk(v, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) == 0 {
			return v, nil
		}

		// Check if this key should be renamed.
		for oldKey, newKey := range rename {
			if p[0].Key() != oldKey {
				continue
			}

			// Add the new key to the accumulator.
			p[0] = dyn.Key(newKey)
			acc, err = dyn.SetByPath(acc, p, v)
			if err != nil {
				return dyn.InvalidValue, err
			}
			return dyn.InvalidValue, dyn.ErrDrop
		}

		// Pass through all other values.
		return v, dyn.ErrSkip
	})
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Merge the accumulator with the original value.
	return merge.Merge(nv, acc)
}
