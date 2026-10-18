package config

import (
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// PermissionList is the permissions of a resource. Their types differ between
// resources, but all have the fields of [resources.PermissionFields].
type PermissionList interface {
	// IsNil reports whether the permissions are not set.
	IsNil() bool
	Len() int
	// At returns permission i.
	At(i int) resources.Permission
	// Append appends p.
	Append(p resources.Permission)
	// Set replaces the permissions with ps.
	Set(ps []resources.Permission)
}

// permissionList is the [PermissionList] of a resource's permissions of type []P.
type permissionList[P ~resources.PermissionFields[L], L ~string] struct {
	p *[]P
}

func (l permissionList[P, L]) IsNil() bool {
	return *l.p == nil
}

func (l permissionList[P, L]) Len() int {
	return len(*l.p)
}

func (l permissionList[P, L]) At(i int) resources.Permission {
	f := resources.PermissionFields[L]((*l.p)[i])
	return resources.Permission{
		Level:                iam.PermissionLevel(f.Level),
		UserName:             f.UserName,
		ServicePrincipalName: f.ServicePrincipalName,
		GroupName:            f.GroupName,
	}
}

func (l permissionList[P, L]) Set(ps []resources.Permission) {
	*l.p = make([]P, 0, len(ps))
	for _, p := range ps {
		l.Append(p)
	}
}

func (l permissionList[P, L]) Append(p resources.Permission) {
	*l.p = append(*l.p, P(resources.PermissionFields[L]{
		Level:                L(p.Level),
		UserName:             p.UserName,
		ServicePrincipalName: p.ServicePrincipalName,
		GroupName:            p.GroupName,
	}))
}

// sortedKeys returns the keys of a resource map in iteration order.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

// ResourceNodes returns the keys of all resources and of their permissions and grants
// where those are present ("resources.jobs.foo", "resources.jobs.foo.permissions"):
// first all resources, then all permissions, then all grants. Permissions and grants
// are present if they are set or hold a pure reference.
func (r *Root) ResourceNodes() []string {
	var nodes []string
	r.Resources.IterateAllResources(func(typ, name string, _ ConfigResource) bool {
		nodes = append(nodes, structpath.NewPath(nil, "resources", typ, name).String())
		return true
	})
	r.Resources.IterateAllPermissions(func(typ, name string, perms PermissionList) bool {
		p := structpath.NewPath(nil, "resources", typ, name, "permissions")
		if !perms.IsNil() || r.IsReferenceAt(p) {
			nodes = append(nodes, p.String())
		}
		return true
	})
	r.Resources.IterateAllGrants(func(typ, name string, grants *[]catalog.PrivilegeAssignment) bool {
		p := structpath.NewPath(nil, "resources", typ, name, "grants")
		if *grants != nil || r.IsReferenceAt(p) {
			nodes = append(nodes, p.String())
		}
		return true
	})
	return nodes
}
