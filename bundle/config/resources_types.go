package config

import (
	"reflect"

	"github.com/databricks/cli/libs/structdiff/structtag"
)

// ResourcesTypes maps the configuration key of each Databricks resource group (for example
// "jobs" or "pipelines") to the Go type that represents a single resource instance inside
// that group (for example `resources.Job`).
var ResourcesTypes = func() map[string]reflect.Type {
	var r Resources
	rt := reflect.TypeOf(r)
	res := make(map[string]reflect.Type, rt.NumField())

	for _, field := range reflect.VisibleFields(rt) {
		tag := structtag.JSONTag(field.Tag.Get("json"))
		name := tag.Name()
		if name == "" || name == "-" {
			continue
		}

		// The type stored in Resources fields is expected to be:
		// map[string]*resources.SomeType
		if field.Type.Kind() != reflect.Map {
			continue
		}
		elemType := field.Type.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}

		res[name] = elemType
	}

	return res
}()
