package main

import (
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
)

// pathMapping provides bidirectional mapping between Go type paths
// (e.g., "github.com/databricks/cli/bundle/config.Bundle") and bundle paths
// (e.g., "bundle"). This allows annotations.yml to use human-readable bundle
// paths instead of Go import paths.
type pathMapping struct {
	// Go type path -> bundle path (e.g., "github.com/.../config.Bundle" -> "bundle")
	typeToBundlePath map[string]string
	// bundle path -> Go type path
	bundlePathToType map[string]string
	// Track visited types to avoid infinite recursion.
	visited map[string]bool
}

// buildPathMapping walks config.Root via reflection to build mappings between
// Go type paths and bundle paths. It skips the "targets" and "environments"
// fields since they mirror the root structure.
func buildPathMapping() *pathMapping {
	m := &pathMapping{
		typeToBundlePath: map[string]string{},
		bundlePathToType: map[string]string{},
		visited:          map[string]bool{},
	}

	rootType := reflect.TypeOf(config.Root{})
	m.typeToBundlePath[getPath(rootType)] = "root"
	m.bundlePathToType["root"] = getPath(rootType)

	m.walkType(rootType, "")
	return m
}

// walkType recursively walks a Go type and records the bundle path for each
// named struct type encountered.
func (m *pathMapping) walkType(typ reflect.Type, currentPath string) {
	typ = derefPtr(typ)

	if typ.Kind() != reflect.Struct {
		return
	}

	typPath := getPath(typ)
	if m.visited[typPath] {
		return
	}
	m.visited[typPath] = true

	for i := range typ.NumField() {
		field := typ.Field(i)

		// Skip unexported fields.
		if !field.IsExported() {
			continue
		}

		// Anonymous (embedded) fields: walk into them at the current path
		// level since their fields are promoted to the parent struct.
		if field.Anonymous {
			m.walkType(field.Type, currentPath)
			continue
		}

		// Get the JSON field name.
		jsonName := jsonFieldName(field)
		if jsonName == "" || jsonName == "-" {
			continue
		}

		// For targets/environments, record the Target type mapping and
		// walk it to pick up types unique to Target (like TargetVariable).
		// The visited check prevents re-walking types already seen from Root.
		if currentPath == "" && (jsonName == "targets" || jsonName == "environments") {
			elemType := derefPtr(field.Type)
			if elemType.Kind() == reflect.Map {
				valType := derefPtr(elemType.Elem())
				if valType.Name() != "" {
					targetPath := jsonName + ".*"
					m.recordMapping(getPath(valType), targetPath)
					m.walkType(valType, targetPath)
				}
			}
			continue
		}

		fieldPath := jsonName
		if currentPath != "" {
			fieldPath = currentPath + "." + jsonName
		}

		fieldType := derefPtr(field.Type)

		switch fieldType.Kind() {
		case reflect.Map:
			// For map types like map[string]*resources.Job, the path includes "*"
			// for the map value type.
			elemType := derefPtr(fieldType.Elem())
			elemPath := fieldPath + ".*"
			if elemType.Name() != "" {
				m.recordMapping(getPath(elemType), elemPath)
			}
			if elemType.Kind() == reflect.Struct {
				m.walkType(elemType, elemPath)
			}

		case reflect.Slice:
			elemType := derefPtr(fieldType.Elem())
			elemPath := fieldPath + ".*"
			if elemType.Name() != "" {
				m.recordMapping(getPath(elemType), elemPath)
			}
			if elemType.Kind() == reflect.Struct {
				m.walkType(elemType, elemPath)
			}

		case reflect.Struct:
			if fieldType.Name() != "" {
				m.recordMapping(getPath(fieldType), fieldPath)
			}
			m.walkType(fieldType, fieldPath)

		default:
			// Record non-struct named types (e.g., string enums).
			if fieldType.Name() != "" && fieldType.PkgPath() != "" {
				m.recordMapping(getPath(fieldType), fieldPath)
			}
		}
	}
}

// recordMapping records a type-to-path mapping. If a type already has a mapping,
// the shorter (higher-level) path wins.
func (m *pathMapping) recordMapping(typePath, bundlePath string) {
	if existing, ok := m.typeToBundlePath[typePath]; ok {
		// Keep the shorter path as canonical.
		if len(existing) <= len(bundlePath) {
			return
		}
	}
	m.typeToBundlePath[typePath] = bundlePath
	m.bundlePathToType[bundlePath] = typePath
}

// derefPtr dereferences pointer types to their base type.
func derefPtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// jsonFieldName extracts the JSON field name from a struct field's json tag.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}
