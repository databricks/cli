package jsonloader

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/dyn"
)

type loader struct {
}

func newLoader() *loader {
	return &loader{}
}

func errorf(loc dyn.Location, format string, args ...interface{}) error {
	return fmt.Errorf("json (%s): %s", loc, fmt.Sprintf(format, args...))
}

func (d *loader) load(node any, loc dyn.Location) (dyn.Value, error) {
	var value dyn.Value
	var err error

	if node == nil {
		return dyn.NilValue, nil
	}

	if reflect.TypeOf(node).Kind() == reflect.Ptr {
		return d.load(reflect.ValueOf(node).Elem().Interface(), loc)
	}

	switch reflect.TypeOf(node).Kind() {
	case reflect.Map:
		value, err = d.loadMapping(node.(map[string]interface{}), loc)
	case reflect.Slice:
		value, err = d.loadSequence(node.([]interface{}), loc)
	case reflect.String, reflect.Bool,
		reflect.Float64, reflect.Float32,
		reflect.Int, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint32, reflect.Uint64:
		value, err = d.loadScalar(node, loc)

	default:
		return dyn.InvalidValue, errorf(loc, "unknown node kind: %v", reflect.TypeOf(node).Kind())
	}

	if err != nil {
		return dyn.InvalidValue, err
	}

	return value, nil
}

func (d *loader) loadScalar(node any, loc dyn.Location) (dyn.Value, error) {
	switch reflect.TypeOf(node).Kind() {
	case reflect.String:
		return dyn.NewValue(node.(string), []dyn.Location{loc}), nil
	case reflect.Bool:
		return dyn.NewValue(node.(bool), []dyn.Location{loc}), nil
	case reflect.Float64, reflect.Float32:
		return dyn.NewValue(node.(float64), []dyn.Location{loc}), nil
	case reflect.Int, reflect.Int32, reflect.Int64:
		return dyn.NewValue(node.(int64), []dyn.Location{loc}), nil
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		return dyn.NewValue(node.(uint64), []dyn.Location{loc}), nil
	default:
		return dyn.InvalidValue, errorf(loc, "unknown scalar type: %v", reflect.TypeOf(node).Kind())
	}
}

func (d *loader) loadSequence(node []interface{}, loc dyn.Location) (dyn.Value, error) {
	dst := make([]dyn.Value, len(node))
	for i, value := range node {
		v, err := d.load(value, loc)
		if err != nil {
			return dyn.InvalidValue, err
		}
		dst[i] = v
	}
	return dyn.NewValue(dst, []dyn.Location{loc}), nil
}

func (d *loader) loadMapping(node map[string]interface{}, loc dyn.Location) (dyn.Value, error) {
	dst := make(map[string]dyn.Value)
	index := 0
	for key, value := range node {
		index += 1
		v, err := d.load(value, dyn.Location{
			Line:   loc.Line + index,
			Column: loc.Column,
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
		dst[key] = v
	}
	return dyn.NewValue(dst, []dyn.Location{loc}), nil
}
