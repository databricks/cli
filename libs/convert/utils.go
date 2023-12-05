package convert

import (
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/convert"
)

var skipFields = []string{"ForceSendFields"}

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
	return key
}

// This struct is used to generate indexes for ordering of map keys.
// The ordering defined based on any predefined order in `order` field
// or running order based on `index`
type order struct {
	index int
	order []string
}

func newOrder(o []string) *order {
	return &order{index: 0, order: o}
}

// Returns an integer which represents the order of map key in resulting
// The lower the index, the earlier in the list the key is.
// If the order is not predefined, it uses running order and any subsequential call to
// order.get returns an increasing index.
func (o *order) get(key string) int {
	index := slices.Index(o.order, key)
	// If the key is found in predefined order list
	// We return a negative index which put the value at the top of the order compared to other
	// not predefined keys. The earlier value in predefined list, the lower negative index value
	if index != -1 {
		return index - len(o.order)
	}

	// Otherwise we just increase the order index
	o.index += 1
	return o.index
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
