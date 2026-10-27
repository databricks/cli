package merge

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// Select returns a new map that contains only the keys specified in the included list.
func Select(value dyn.Value, included []string) (dyn.Value, error) {
	mapping, ok := value.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected a map, but found %s", value.Kind())
	}

	newMapping := dyn.NewMapping()
	for _, key := range included {
		pair, ok := mapping.GetPairByString(key)

		if ok {
			newMapping.SetLoc(key, pair.Key.Locations(), pair.Value)
		}
	}

	return dyn.NewValue(newMapping, value.Locations()), nil
}

// AntiSelect returns a new map with all keys from the input map except for the ones in the excluded list.
func AntiSelect(value dyn.Value, excluded []string) (dyn.Value, error) {
	mapping, ok := value.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected a map, but found %s", value.Kind())
	}

	excludedSet := make(map[string]struct{})
	for _, key := range excluded {
		excludedSet[key] = struct{}{}
	}

	included := make([]string, 0, len(mapping.Pairs()))
	for _, pair := range mapping.Pairs() {
		key, ok := pair.Key.AsString()
		if !ok {
			return dyn.InvalidValue, fmt.Errorf("expected a string key, but found %s", pair.Key.Kind())
		}

		if _, ok := excludedSet[key]; !ok {
			included = append(included, pair.Key.MustString())
		}
	}

	return Select(value, included)
}
