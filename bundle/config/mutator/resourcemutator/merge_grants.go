package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
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
	var err error
	b.Config.Resources.IterateAllGrants(func(typ, name string, grants *[]catalog.PrivilegeAssignment) bool {
		err = mergeGrantsOf(b, typ, name, grants)
		return err == nil
	})
	return diag.FromErr(err)
}

// mergeGrantsOf merges the grants of the resource name of type typ by principal, and
// deduplicates the privileges of each grant.
func mergeGrantsOf(b *bundle.Bundle, typ, name string, grants *[]catalog.PrivilegeAssignment) error {
	if len(*grants) == 0 {
		return nil
	}

	// Merge grant entries by principal. This concatenates privileges
	// for entries with the same principal via the standard merge semantics.
	path := structpath.NewPath(nil, "resources", typ, name, "grants")
	err := b.Config.MergeElementsByKey(path, "principal", func(v structvar.View) string {
		s, _ := v.AsString()
		return s
	}, false)
	if err != nil {
		return err
	}

	// Deduplicate privileges within each grant entry (the merge replaced the slice).
	gs := *grants
	for i := range gs {
		privileges, sources := deduplicatePrivileges(gs[i].Privileges)
		if len(privileges) == len(gs[i].Privileges) {
			continue
		}
		gs[i].Privileges = privileges
		b.Config.UpdateSequence(structpath.NewPath(path, i, "privileges"), sources)
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
