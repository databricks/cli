package configsync

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// ensurePathExists ensures all intermediate nodes exist in the path.
// It creates empty maps for missing intermediate map keys.
// For sequence indices, it verifies they exist but does not create them.
// Returns the modified value with all intermediate nodes guaranteed to exist.
func ensurePathExists(v dyn.Value, path dyn.Path) (dyn.Value, error) {
	if len(path) == 0 {
		return v, nil
	}

	result := v
	for i := 1; i < len(path); i++ {
		prefixPath := path[:i]
		component := path[i-1]

		item, _ := dyn.GetByPath(result, prefixPath)
		if !item.IsValid() {
			if component.Key() != "" {
				if i < len(path) && path[i].Key() == "" {
					return dyn.InvalidValue, fmt.Errorf("sequence index does not exist at path %s", prefixPath)
				}

				var err error
				result, err = dyn.SetByPath(result, prefixPath, dyn.V(dyn.NewMapping()))
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to create intermediate path %s: %w", prefixPath, err)
				}
			} else {
				return dyn.InvalidValue, fmt.Errorf("sequence index does not exist at path %s", prefixPath)
			}
		}
	}

	return result, nil
}
