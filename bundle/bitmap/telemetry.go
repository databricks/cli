package bitmap

import (
	"fmt"
	"reflect"
	"unicode"
)

// telemetryPrefix namespaces the internal telemetry struct's fields in the
// schema so they never collide with config.Root paths.
const telemetryPrefix = "telemetry"

// boolValues is implemented by bundle.BoolSet, a telemetry field that records
// each value it took. It is matched by method so that this package does not
// import the bundle package.
type boolValues interface{ Values() []bool }

var boolValuesType = reflect.TypeFor[boolValues]()

// telemetryPaths returns the schema entries of the telemetry struct: each field
// under its name (see telemetryPath), followed by the entries of the values it can
// record. A plain bool is recorded only when true, so its presence bit is its value.
func telemetryPaths(telemetryType reflect.Type) ([]string, error) {
	var paths []string
	fields := map[string]string{}
	for field := range telemetryType.Fields() {
		path := telemetryPath(field)
		if other, ok := fields[path]; ok {
			return nil, fmt.Errorf("telemetry fields %s and %s are both named %s", other, field.Name, path)
		}
		fields[path] = field.Name
		paths = append(paths, path)
		if field.Type.Kind() != reflect.Bool {
			paths = append(paths, path+trueSuffix, path+falseSuffix)
		}
	}
	return paths, nil
}

// telemetryPath names a telemetry field's bits: its metric tag (its key in the
// legacy bool_values) when it has one, else its Go name in snake case. Bits are
// decoded by name, so renaming a field without a metric tag renames its bits.
func telemetryPath(f reflect.StructField) string {
	name := f.Tag.Get("metric")
	if name == "" {
		name = snakeCase(f.Name)
	}
	return joinField(telemetryPrefix, name)
}

// snakeCase converts a Go name to snake case, keeping acronyms together:
// "DMSCompatAuto" becomes "dms_compat_auto".
func snakeCase(name string) string {
	runes := []rune(name)
	var out []rune
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if !unicode.IsUpper(prev) || nextLower {
				out = append(out, '_')
			}
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

// markTelemetry sets the bits of the measured telemetry fields: the presence
// bit and the bit of each value taken. A plain bool counts only when true.
func markTelemetry(telemetry any, index map[string]int, bits []bool) error {
	v := reflect.ValueOf(telemetry)
	for i := range v.NumField() {
		field := v.Type().Field(i)
		path := telemetryPath(field)
		var values []bool
		switch f := v.Field(i).Interface().(type) {
		case bool:
			if f {
				setBit(index, bits, path)
			}
			continue
		case *bool:
			if f != nil {
				values = []bool{*f}
			}
		case boolValues:
			values = f.Values()
		default:
			return fmt.Errorf("telemetry field %s has unsupported type %s", field.Name, field.Type)
		}
		for _, value := range values {
			setBit(index, bits, path)
			setBit(index, bits, path+valueSuffix(value))
		}
	}
	return nil
}
