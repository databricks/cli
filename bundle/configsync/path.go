package configsync

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// ensurePathExists ensures all intermediate nodes exist in the path.
// It creates empty maps for missing intermediate map keys.
// For sequences, it creates empty sequences with empty map elements when needed.
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
				key := path[i].Key()
				isIndex := key == ""
				isKey := key != ""

				if i < len(path) && isIndex {
					index := path[i].Index()
					seq := make([]dyn.Value, index+1)
					for j := range seq {
						seq[j] = dyn.V(dyn.NewMapping())
					}
					var err error
					result, err = dyn.SetByPath(result, prefixPath, dyn.V(seq))
					if err != nil {
						return dyn.InvalidValue, fmt.Errorf("failed to create sequence at path %s: %w", prefixPath, err)
					}
				} else if isKey {
					var err error
					result, err = dyn.SetByPath(result, prefixPath, dyn.V(dyn.NewMapping()))
					if err != nil {
						return dyn.InvalidValue, fmt.Errorf("failed to create intermediate path %s: %w", prefixPath, err)
					}
				}
			} else {
				return dyn.InvalidValue, fmt.Errorf("sequence index does not exist at path %s", prefixPath)
			}
		}
	}

	return result, nil
}
