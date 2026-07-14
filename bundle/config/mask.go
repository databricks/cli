package config

import (
	"fmt"
	"sync"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

const sensitiveValueMask = "********"

// sensitiveFieldsCache caches sensitive field names per resource type key.
var (
	sensitiveFieldsCache     map[string]map[string]bool
	sensitiveFieldsCacheOnce = sync.OnceFunc(func() {
		sensitiveFieldsCache = make(map[string]map[string]bool)
		for name, typ := range ResourcesTypes {
			if fields := convert.SensitiveFieldNames(typ); len(fields) > 0 {
				sensitiveFieldsCache[name] = fields
			}
		}
	})
)

// SensitiveFields returns a map of JSON field names → true for a resource type
// key (e.g. "secrets"). Built once from the convert.SensitiveFieldNames helper.
func SensitiveFields(resourceTypeKey string) map[string]bool {
	sensitiveFieldsCacheOnce()
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
		fields := SensitiveFields(resourceType)
		if len(fields) == 0 {
			return resource, nil
		}

		for fieldName := range fields {
			fv, err := dyn.GetByPath(resource, dyn.NewPath(dyn.Key(fieldName)))
			if dyn.IsNoSuchKeyError(err) {
				// Field not present in this resource instance — nothing to mask.
				continue
			}
			if err != nil {
				return dyn.InvalidValue, err
			}
			if fv.Kind() == dyn.KindNil {
				continue
			}
			s, ok := fv.AsString()
			if !ok {
				return dyn.InvalidValue, fmt.Errorf("sensitive field %q must be a string, got %s", fieldName, fv.Kind())
			}
			if s == "" {
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
