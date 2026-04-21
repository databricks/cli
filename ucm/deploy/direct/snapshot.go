package direct

import (
	"github.com/databricks/cli/ucm/config/resources"
)

// Snapshot-helpers live in their own file so plan.go (the diff logic) stays
// focused on the Create/Update/Skip/Delete decision tree. Each helper takes
// a pointer into the loaded ucm config and returns the comparable state
// representation.

func catalogStateFromConfig(c *resources.Catalog) CatalogState {
	if c == nil {
		return CatalogState{}
	}
	return CatalogState{
		Name:        c.Name,
		Comment:     c.Comment,
		StorageRoot: c.StorageRoot,
		Tags:        copyTags(c.Tags),
	}
}

func schemaStateFromConfig(s *resources.Schema) SchemaState {
	if s == nil {
		return SchemaState{}
	}
	return SchemaState{
		Name:    s.Name,
		Catalog: s.Catalog,
		Comment: s.Comment,
		Tags:    copyTags(s.Tags),
	}
}

func grantStateFromConfig(g *resources.Grant) GrantState {
	if g == nil {
		return GrantState{}
	}
	privs := make([]string, len(g.Privileges))
	copy(privs, g.Privileges)
	return GrantState{
		SecurableType: g.Securable.Type,
		SecurableName: g.Securable.Name,
		Principal:     g.Principal,
		Privileges:    privs,
	}
}

func copyTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}
