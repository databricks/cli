package yamlsaver

import (
	"reflect"
	"strings"
)

// Returns config name to be used in YAML configuration for
// the config value passed. Uses the name defined in 'json' tag
// for the structure. If the name is not defined or
// there is no 'json' tag defined, it returns the field name itself.
// Second return value is true if the name was defined in 'json' tag.
func ConfigKey(v any, name string) (string, bool) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = reflect.TypeOf(v).Elem()
	}
	field, ok := t.FieldByName(name)
	if !ok {
		return name, false
	}
	key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if key == "-" || key == "" {
		return name, false
	}
	return key, true
}
