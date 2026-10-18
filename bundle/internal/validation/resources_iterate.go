package main

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

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

	out := &g.out
	fmt.Fprintf(out, "\n// IterateAllResources calls fn for every resource, in the order of the resource type\n// and then of the name, until fn returns false. Resources defined as null are\n// visited with a nil res.\n")
	fmt.Fprintf(out, "func (r *Resources) IterateAllResources(fn func(typ, name string, res ConfigResource) bool) {\n")
	for _, t := range types {
		fmt.Fprintf(out, "\tfor _, name := range sortedKeys(r.%s) {\n\t\tvar res ConfigResource\n\t\tif x := r.%s[name]; x != nil {\n\t\t\tres = x\n\t\t}\n\t\tif !fn(%q, name, res) {\n\t\t\treturn\n\t\t}\n\t}\n", t.field, t.field, t.key)
	}
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "\n// IterateAllPermissions calls fn for the permissions of every resource that can have\n// them, in the order of IterateAllResources, until fn returns false.\n")
	fmt.Fprintf(out, "func (r *Resources) IterateAllPermissions(fn func(typ, name string, perms PermissionList) bool) {\n")
	for _, t := range types {
		if t.perms == "" {
			continue
		}
		fmt.Fprintf(out, "\tfor _, name := range sortedKeys(r.%s) {\n\t\tif x := r.%s[name]; x != nil && !fn(%q, name, permissionList[%s, %s]{&x.%s}) {\n\t\t\treturn\n\t\t}\n\t}\n",
			t.field, t.field, t.key, g.typeName(t.permElem), g.typeName(t.permElem.Field(0).Type), t.perms)
	}
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "\n// IterateAllResourceStates calls fn for the deployed id and modified status of every\n// resource that has them, in the order of IterateAllResources, until fn returns false.\n")
	fmt.Fprintf(out, "func (r *Resources) IterateAllResourceStates(fn func(typ, name string, id *string, status *%s) bool) {\n", g.typeName(reflect.TypeFor[resources.ModifiedStatus]()))
	for _, t := range types {
		if t.id == "" || !t.status {
			continue
		}
		fmt.Fprintf(out, "\tfor _, name := range sortedKeys(r.%s) {\n\t\tif x := r.%s[name]; x != nil && !fn(%q, name, &x.%s, &x.ModifiedStatus) {\n\t\t\treturn\n\t\t}\n\t}\n", t.field, t.field, t.key, t.id)
	}
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, "\n// GetOrCreateState returns the deployed id and modified status of the resource name of\n// type typ, creating the resource if it does not exist (created is then true). ok is\n// false if resources of type typ have no state.\n")
	fmt.Fprintf(out, "func (r *Resources) GetOrCreateState(typ, name string) (id *string, status *%s, created, ok bool) {\n\tswitch typ {\n", g.typeName(reflect.TypeFor[resources.ModifiedStatus]()))
	for _, t := range types {
		if t.id == "" || !t.status {
			continue
		}
		fmt.Fprintf(out, "\tcase %q:\n\t\tx, exists := r.%s[name]\n\t\tif x == nil {\n\t\t\tif r.%s == nil {\n\t\t\t\tr.%s = map[string]*%s{}\n\t\t\t}\n\t\t\tx = &%s{}\n\t\t\tr.%s[name] = x\n\t\t}\n\t\treturn &x.%s, &x.ModifiedStatus, !exists, true\n",
			t.key, t.field, t.field, t.field, g.typeName(t.elem), g.typeName(t.elem), t.field, t.id)
	}
	fmt.Fprintf(out, "\tdefault:\n\t\treturn nil, nil, false, false\n\t}\n}\n")

	fmt.Fprintf(out, "\n// IsZero reports whether no resource map is set.\n")
	fmt.Fprintf(out, "func (r *Resources) IsZero() bool {\n\treturn ")
	for i, t := range types {
		if i > 0 {
			fmt.Fprintf(out, " &&\n\t\t")
		}
		fmt.Fprintf(out, "r.%s == nil", t.field)
	}
	fmt.Fprintf(out, "\n}\n")

	fmt.Fprintf(out, "\n// IterateAllGrants calls fn for the grants of every resource that can have them, in\n// the order of IterateAllResources, until fn returns false.\n")
	fmt.Fprintf(out, "func (r *Resources) IterateAllGrants(fn func(typ, name string, grants *[]%s) bool) {\n", g.typeName(grantsType.Elem()))
	for _, t := range types {
		if t.grants == "" {
			continue
		}
		fmt.Fprintf(out, "\tfor _, name := range sortedKeys(r.%s) {\n\t\tif x := r.%s[name]; x != nil && !fn(%q, name, &x.%s) {\n\t\t\treturn\n\t\t}\n\t}\n", t.field, t.field, t.key, t.grants)
	}
	fmt.Fprintf(out, "}\n")

	var src bytes.Buffer
	fmt.Fprintf(&src, "// Code generated by bundle/internal/validation. DO NOT EDIT.\n\npackage config\n\nimport (\n")
	paths := make([]string, 0, len(g.imports))
	for p := range g.imports {
		if p != "github.com/databricks/cli/bundle/config" {
			paths = append(paths, p)
		}
	}
	slices.SortFunc(paths, cmp.Compare)
	for _, p := range paths {
		if path.Base(p) == g.imports[p] {
			fmt.Fprintf(&src, "\t%q\n", p)
		} else {
			fmt.Fprintf(&src, "\t%s %q\n", g.imports[p], p)
		}
	}
	fmt.Fprintf(&src, ")\n")
	src.Write(out.Bytes())
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		_ = os.WriteFile(file+".broken", src.Bytes(), 0o644)
		return fmt.Errorf("failed to format %s: %w", file, err)
	}
	return os.WriteFile(file, formatted, 0o644)
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
