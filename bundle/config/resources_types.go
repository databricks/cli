package config

import (
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structtag"
)

// resourceFieldName returns the configuration key for a resource field.
func resourceFieldName(field reflect.StructField) (string, bool) {
	if structaccess.IsSkippedField(field) {
		return "", false
	}
	name := structtag.JSONTag(field.Tag.Get("json")).Name()
	return name, name != ""
}

// ResourcesTypes maps the configuration key of each Databricks resource group (for example
// "jobs" or "pipelines") to the Go type that represents a single resource instance inside
// that group (for example `resources.Job`).

var ResourcesTypes = func() map[string]reflect.Type {
	rt := reflect.TypeFor[Resources]()
	res := make(map[string]reflect.Type, rt.NumField())

	for _, field := range reflect.VisibleFields(rt) {
		name, ok := resourceFieldName(field)
		if !ok {
			continue
		}

		// The type stored in Resources fields is expected to be:
		// map[string]*resources.SomeType
		if field.Type.Kind() != reflect.Map {
			continue
		}
		elemType := field.Type.Elem()
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}

		res[name] = elemType

		// Automatically detect and add permissions field types
		// Look for a "Permissions" field in the resource type
		for _, resourceField := range reflect.VisibleFields(elemType) {
			if resourceField.Name == "Permissions" {
				permissionsKey := name + ".permissions"
				res[permissionsKey] = resourceField.Type
				continue
			}
			if resourceField.Name == "Grants" {
				grantsKey := name + ".grants"
				res[grantsKey] = resourceField.Type
			}
		}
	}

	return res
}()
