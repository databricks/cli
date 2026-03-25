package tableview

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/databricks/databricks-sdk-go/listing"
)

const maxAutoColumns = 8

var autoCache sync.Map // reflect.Type -> *TableConfig

// AutoDetect creates a TableConfig by reflecting on the element type of the iterator.
// It picks up to maxAutoColumns top-level scalar fields.
// Returns nil if no suitable columns are found.
func AutoDetect[T any](iter listing.Iterator[T]) *TableConfig {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if cached, ok := autoCache.Load(t); ok {
		return cached.(*TableConfig)
	}

	cfg := autoDetectFromType(t)
	if cfg != nil {
		autoCache.Store(t, cfg)
	}
	return cfg
}

func autoDetectFromType(t reflect.Type) *TableConfig {
	if t.Kind() != reflect.Struct {
		return nil
	}

	var columns []ColumnDef
	for i := range t.NumField() {
		if len(columns) >= maxAutoColumns {
			break
		}
		field := t.Field(i)
		if !field.IsExported() || field.Anonymous {
			continue
		}
		if !isScalarKind(field.Type.Kind()) {
			continue
		}

		header := fieldHeader(field)
		columns = append(columns, ColumnDef{
			Header: header,
			Extract: func(v any) string {
				val := reflect.ValueOf(v)
				if val.Kind() == reflect.Ptr {
					if val.IsNil() {
						return ""
					}
					val = val.Elem()
				}
				if val.Kind() != reflect.Struct {
					return ""
				}
				f := val.Field(i)
				return fmt.Sprintf("%v", f.Interface())
			},
		})
	}

	if len(columns) == 0 {
		return nil
	}
	return &TableConfig{Columns: columns}
}

func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// fieldHeader converts a struct field to a display header.
// Uses the json tag if available, otherwise the field name.
func fieldHeader(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag != "" {
		name, _, _ := strings.Cut(tag, ",")
		if name != "" && name != "-" {
			return snakeToTitle(name)
		}
	}
	return f.Name
}

func snakeToTitle(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if w == "id" {
			words[i] = "ID"
		} else if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
