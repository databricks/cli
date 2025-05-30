package config

import (
	"reflect"

	"github.com/databricks/cli/libs/structdiff/jsontag"
)

// ResourcesTypes maps the configuration key of each Databricks resource section (for example
// "jobs" or "pipelines") to the Go type that represents a single resource instance inside
// that section (for example `resources.Job`).
//
// The map is populated at package‐initialisation time by inspecting the `Resources` struct with
// reflection, so it automatically stays up-to-date when new resource kinds are added.
var ResourcesTypes = func() map[string]reflect.Type {
	var r Resources
	rt := reflect.TypeOf(r)
	res := make(map[string]reflect.Type, rt.NumField())

	for _, field := range reflect.VisibleFields(rt) {
		tag := jsontag.JSONTag(field.Tag.Get("json"))
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
