package resourcemutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type mergeGrants struct{}

// MergeGrants returns a mutator that deduplicates grant entries.
// It merges entries with the same principal and deduplicates privileges.
func MergeGrants() bundle.Mutator {
	return &mergeGrants{}
}

func (m *mergeGrants) Name() string {
	return "MergeGrants"
}

func (m *mergeGrants) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	r := &b.Config.Resources

	// Resource types that support grants.
	err := mergeGrantsOf(b, "catalogs", r.Catalogs, func(x *resources.Catalog) []catalog.PrivilegeAssignment { return x.Grants })
	if err == nil {
		err = mergeGrantsOf(b, "schemas", r.Schemas, func(x *resources.Schema) []catalog.PrivilegeAssignment { return x.Grants })
	}
	if err == nil {
		err = mergeGrantsOf(b, "external_locations", r.ExternalLocations, func(x *resources.ExternalLocation) []catalog.PrivilegeAssignment { return x.Grants })
	}
	if err == nil {
		err = mergeGrantsOf(b, "secrets", r.Secrets, func(x *resources.Secret) []catalog.PrivilegeAssignment { return x.Grants })
	}
	if err == nil {
		err = mergeGrantsOf(b, "volumes", r.Volumes, func(x *resources.Volume) []catalog.PrivilegeAssignment { return x.Grants })
	}
	if err == nil {
		err = mergeGrantsOf(b, "registered_models", r.RegisteredModels, func(x *resources.RegisteredModel) []catalog.PrivilegeAssignment { return x.Grants })
	}
	if err == nil {
		err = mergeGrantsOf(b, "vector_search_indexes", r.VectorSearchIndexes, func(x *resources.VectorSearchIndex) []catalog.PrivilegeAssignment { return x.Grants })
	}
	return diag.FromErr(err)
}

// mergeGrantsOf merges the grants of every resource of the given type (visited in sorted
// order) by principal, and deduplicates the privileges of each grant.
func mergeGrantsOf[R any](b *bundle.Bundle, resourceType string, rs map[string]*R, grants func(*R) []catalog.PrivilegeAssignment) error {
	names := make([]string, 0, len(rs))
	for name := range rs {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		if rs[name] == nil || len(grants(rs[name])) == 0 {
			continue
		}

		// Merge grant entries by principal. This concatenates privileges
		// for entries with the same principal via the standard merge semantics.
		path := structpath.NewPath(nil, "resources", resourceType, name, "grants")
		err := b.Config.MergeElementsByKey(path, "principal", func(v structvar.View) string {
			s, _ := v.AsString()
			return s
		}, false)
		if err != nil {
			return err
		}

		// Deduplicate privileges within each grant entry.
		gs := grants(rs[name])
		for i := range gs {
			privileges, sources := deduplicatePrivileges(gs[i].Privileges)
			if len(privileges) == len(gs[i].Privileges) {
				continue
			}
			gs[i].Privileges = privileges
			b.Config.UpdateSequence(structpath.NewPath(path, i, "privileges"), sources)
		}
	}
	return nil
}

// deduplicatePrivileges removes duplicate privileges, preserving the order of first
// appearance. It also returns, for every remaining privilege, its index in the input.
func deduplicatePrivileges(in []catalog.Privilege) ([]catalog.Privilege, [][]int) {
	seen := make(map[catalog.Privilege]bool, len(in))
	out := make([]catalog.Privilege, 0, len(in))
	sources := make([][]int, 0, len(in))
	for i, p := range in {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		sources = append(sources, []int{i})
	}
	return out, sources
}
