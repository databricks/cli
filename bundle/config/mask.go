package config

import (
	"sync"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

const sensitiveValueMask = "********"

// sensitiveFieldsCache caches sensitive field names per resource type key.
var (
	sensitiveFieldsCache     map[string]map[string]bool
	sensitiveFieldsCacheOnce sync.Once
)

// sensitiveFields returns a map of JSON field names → true for resource type
// key (e.g. "secrets"). Built once from the convert.SensitiveFieldNames helper.
func sensitiveFields(resourceTypeKey string) map[string]bool {
	sensitiveFieldsCacheOnce.Do(func() {
		sensitiveFieldsCache = make(map[string]map[string]bool)
		for name, typ := range ResourcesTypes {
			if fields := convert.SensitiveFieldNames(typ); len(fields) > 0 {
				sensitiveFieldsCache[name] = fields
			}
		}
	})
	return sensitiveFieldsCache[resourceTypeKey]
}

// MaskSensitiveFields returns a copy of v with all fields tagged
// `bundle:"sensitive"` replaced by [sensitiveValueMask].
//
// Only the display copy of a dyn.Value should be passed here —
// the live config value that feeds the deployment pipeline must never be masked.
func MaskSensitiveFields(v dyn.Value) (dyn.Value, error) {
	// Pattern: resources.<type>.<name>
	resourcesPattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.AnyKey(), // resource type (e.g. "secrets")
		dyn.AnyKey(), // resource name
	)

	return dyn.MapByPattern(v, resourcesPattern, func(p dyn.Path, resource dyn.Value) (dyn.Value, error) {
		if len(p) < 2 {
			return resource, nil
		}
		resourceType := p[1].Key()
		fields := sensitiveFields(resourceType)
		if len(fields) == 0 {
			return resource, nil
		}

		for fieldName := range fields {
			fv, err := dyn.GetByPath(resource, dyn.NewPath(dyn.Key(fieldName)))
			if err != nil {
				// Field not present — nothing to mask.
				continue
			}
			s, ok := fv.AsString()
			if !ok || s == "" {
				// Not a non-empty string — nothing to mask.
				continue
			}
			resource, err = dyn.SetByPath(resource, dyn.NewPath(dyn.Key(fieldName)),
				dyn.NewValue(sensitiveValueMask, fv.Locations()))
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return resource, nil
	})
}
