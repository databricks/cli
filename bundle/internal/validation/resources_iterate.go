package main

import (
	"bytes"
	"cmp"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

//go:embed resources_iterate.go.tmpl
var resourcesIterateSource string

var resourcesIterateTemplate = template.Must(template.New("resources_iterate").Parse(resourcesIterateSource))

// generateResourcesIterate writes bundle/config/resources_iterate.go: typed iterators
// over all resources, their permissions and their grants, generated from the fields of
// config.Resources so that new resource types are covered.
func generateResourcesIterate(file string) error {
	g := newGen("", false)
	g.aliases["config"] = true
	rt := reflect.TypeFor[config.Resources]()
	info := structvar.GetStructInfo(rt)

	type resourceType struct {
		key, field string
		elem       reflect.Type // the resource struct
		perms      string       // Go field of the permissions, if any
		permElem   reflect.Type
		grants     string // Go field of the grants, if any
		id         string // Go field of the deployed id, if any
		status     bool   // the resource has a modified_status
	}
	var types []resourceType
	grantsType := reflect.TypeFor[[]catalog.PrivilegeAssignment]()
	for _, f := range info.List {
		ft := rt.FieldByIndex(f.Index).Type
		if ft.Kind() != reflect.Map || ft.Elem().Kind() != reflect.Pointer {
			return fmt.Errorf("resources.%s: expected a map of pointers, got %s", f.Key, ft)
		}
		t := resourceType{key: f.Key, field: rt.FieldByIndex(f.Index).Name, elem: ft.Elem().Elem()}
		ei := structvar.GetStructInfo(t.elem)
		if idx, ok := ei.Fields["permissions"]; ok {
			pf := t.elem.FieldByIndex(idx)
			if !isPermissionsSlice(pf.Type) {
				return fmt.Errorf("resources.%s.permissions: unexpected type %s", f.Key, pf.Type)
			}
			t.perms = pf.Name
			t.permElem = pf.Type.Elem()
		}
		if idx, ok := ei.Fields["grants"]; ok {
			gf := t.elem.FieldByIndex(idx)
			if gf.Type != grantsType {
				return fmt.Errorf("resources.%s.grants: unexpected type %s", f.Key, gf.Type)
			}
			t.grants = gf.Name
		}
		if idx, ok := ei.Fields["id"]; ok && t.elem.FieldByIndex(idx).Type.Kind() == reflect.String {
			t.id = t.elem.FieldByIndex(idx).Name
		}
		_, t.status = ei.Fields["modified_status"]
		types = append(types, t)
	}
	slices.SortFunc(types, func(a, b resourceType) int { return cmp.Compare(a.key, b.key) })

	// Type names register their imports, so the data is built before the header.
	type typeData struct {
		Key, Field, Elem           string
		Perms, PermElem, PermLevel string
		Grants, ID                 string
		HasState                   bool
	}
	data := struct {
		Imports               []string
		Types                 []typeData
		ModifiedStatus, Grant string
	}{
		ModifiedStatus: g.typeName(reflect.TypeFor[resources.ModifiedStatus]()),
		Grant:          g.typeName(grantsType.Elem()),
	}
	for _, t := range types {
		d := typeData{Key: t.key, Field: t.field, Elem: g.typeName(t.elem), Perms: t.perms, Grants: t.grants, ID: t.id, HasState: t.id != "" && t.status}
		if t.perms != "" {
			d.PermElem = g.typeName(t.permElem)
			d.PermLevel = g.typeName(t.permElem.Field(0).Type)
		}
		data.Types = append(data.Types, d)
	}
	for p, alias := range g.imports {
		switch {
		case p == "github.com/databricks/cli/bundle/config":
		case path.Base(p) == alias:
			data.Imports = append(data.Imports, strconv.Quote(p))
		default:
			data.Imports = append(data.Imports, alias+" "+strconv.Quote(p))
		}
	}
	slices.SortFunc(data.Imports, func(a, b string) int { return cmp.Compare(importPath(a), importPath(b)) })

	var src bytes.Buffer
	if err := resourcesIterateTemplate.Execute(&src, data); err != nil {
		return err
	}
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		_ = os.WriteFile(file+".broken", src.Bytes(), 0o644)
		return fmt.Errorf("failed to format %s: %w", file, err)
	}
	return os.WriteFile(file, formatted, 0o644)
}

// importPath returns the quoted path of an import line (with or without an alias).
func importPath(line string) string {
	_, p, _ := strings.Cut(line, `"`)
	return p
}

// isPermissionsSlice reports whether t is a slice of a struct with the fields of
// resources.PermissionFields.
func isPermissionsSlice(t reflect.Type) bool {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Struct {
		return false
	}
	e := t.Elem()
	var names []string
	for i := range e.NumField() {
		names = append(names, e.Field(i).Name)
	}
	return strings.Join(names, ",") == "Level,UserName,ServicePrincipalName,GroupName" && e.Field(0).Type.Kind() == reflect.String
}
