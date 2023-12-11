package convert

import (
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/convert"
)

var skipFields = []string{"ForceSendFields", "Format"}

// Returns config name to be used in YAML configuration for
// the config value passed. Uses the name defined in 'json' tag
// for the structure
func key(v any, name string) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = reflect.TypeOf(v).Elem()
	}
	field, ok := t.FieldByName(name)
	if !ok {
		return name
	}
	key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if key == "-" || key == "" {
		return name
	}
	return key
}

// Converts a struct to map. Skips any nil fields.
// It uses `skipFields` to skip unnecessary fields.
// Uses `order` to define the order of keys in resulting outout
func convertToMapValue(strct any, order *order, dst map[string]config.Value) (config.Value, error) {
	itemValue := reflect.ValueOf(strct)
	if itemValue.Kind() == reflect.Pointer {
		itemValue = itemValue.Elem()
	}
	for i := 0; i < itemValue.NumField(); i++ {
		if !itemValue.Field(i).IsZero() {
			f := itemValue.Type().Field(i)
			if slices.Contains(skipFields, f.Name) {
				continue
			}

			// If the value is already defined in destination, it means it was
			// manually set due to custom ordering or other customisation required
			// So we're skipping processing it again
			if _, ok := dst[key(strct, f.Name)]; ok {
				continue
			}

			ref := config.NilValue
			nv, err := convert.FromTyped(itemValue.Field(i).Interface(), ref)
			if err != nil {
				return config.NilValue, err
			}

			if nv.Kind() != config.KindNil {
				nv.SetLocation(config.Location{Line: order.get(f.Name)})
				dst[key(strct, f.Name)] = nv
			}
		}
	}

	return config.V(dst), nil
}

func replaceNonAlphanumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	return '_'
}

// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including spaces and dots, which are not supported in e.g. experiment names or YAML keys.
func NormaliseString(name string) string {
	name = strings.ToLower(name)
	return strings.Map(replaceNonAlphanumeric, name)
}
