package yamlsaver

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

// Converts a struct to map. Skips any nil fields.
// It uses `skipFields` to skip unnecessary fields.
// Uses `order` to define the order of keys in resulting outout
func ConvertToMapValue(strct any, order *Order, skipFields []string, dst map[string]dyn.Value) (dyn.Value, error) {
	ref := dyn.NilValue
	mv, err := convert.FromTyped(strct, ref)
	if err != nil {
		return dyn.InvalidValue, err
	}

	if mv.Kind() != dyn.KindMap {
		return dyn.InvalidValue, fmt.Errorf("expected map, got %s", mv.Kind())
	}

	mv, err = sortMapAlphabetically(mv)
	if err != nil {
		return dyn.InvalidValue, err
	}

	return skipAndOrder(mv, order, skipFields, dst)
}

// Sort the map alphabetically by keys. This is used to produce stable output for generated YAML files.
func sortMapAlphabetically(mv dyn.Value) (dyn.Value, error) {
	sortedMap := dyn.NewMapping()
	mapV := mv.MustMap()
	keys := mapV.Keys()
	slices.SortStableFunc(keys, func(i, j dyn.Value) int {
		iKey := i.MustString()
		jKey := j.MustString()
		if iKey < jKey {
			return -1
		}

		if iKey > jKey {
			return 1
		}
		return 0
	})

	for _, key := range keys {
		value, _ := mapV.Get(key)
		var err error
		if value.Kind() == dyn.KindMap {
			value, err = sortMapAlphabetically(value)
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		sortedMap.SetLoc(key.MustString(), key.Locations(), value)
	}

	return dyn.V(sortedMap), nil
}

func skipAndOrder(mv dyn.Value, order *Order, skipFields []string, dst map[string]dyn.Value) (dyn.Value, error) {
	for _, pair := range mv.MustMap().Pairs() {
		k := pair.Key.MustString()
		v := pair.Value
		if v.Kind() == dyn.KindNil {
			continue
		}

		if slices.Contains(skipFields, k) {
			continue
		}

		// If the value is already defined in destination, it means it was
		// manually set due to custom ordering or other customisation required
		// So we're skipping processing it again
		if _, ok := dst[k]; ok {
			continue
		}

		if order == nil {
			dst[k] = v
		} else {
			dst[k] = dyn.NewValue(v.Value(), []dyn.Location{{Line: order.Get(k)}})
		}
	}

	return dyn.V(dst), nil
}
