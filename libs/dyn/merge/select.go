package merge

import (
	"fmt"
	"github.com/databricks/cli/libs/dyn"
)

func Select(value dyn.Value, included []string) (dyn.Value, error) {
	mapping, ok := value.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected a map, but found %s", value.Kind())
	}

	newMapping := dyn.NewMapping()
	for _, key := range included {
		pair, ok := mapping.GetPairByString(key)

		if ok {
			err := newMapping.Set(pair.Key, pair.Value)
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
	}

	return dyn.NewValue(newMapping, value.Locations()), nil
}

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
