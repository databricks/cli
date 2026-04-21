package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type inheritCatalogTags struct{}

// InheritCatalogTags merges each catalog's tags into every schema under that
// catalog unless the schema sets tag_inherit: false. Schema tags win on
// conflict. Runs after FlattenNestedResources (so schemas are already flat
// and carry a catalog reference) and before ValidateTags.
func InheritCatalogTags() ucm.Mutator { return &inheritCatalogTags{} }

func (m *inheritCatalogTags) Name() string { return "InheritCatalogTags" }

func (m *inheritCatalogTags) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	catalogs := u.Config.Resources.Catalogs
	for _, schema := range u.Config.Resources.Schemas {
		if schema == nil || schema.Catalog == "" {
			continue
		}
		if schema.TagInherit != nil && !*schema.TagInherit {
			continue
		}
		parent := catalogs[schema.Catalog]
		if parent == nil || len(parent.Tags) == 0 {
			continue
		}
		if schema.Tags == nil {
			schema.Tags = make(map[string]string, len(parent.Tags))
		}
		for k, v := range parent.Tags {
			if _, exists := schema.Tags[k]; !exists {
				schema.Tags[k] = v
			}
		}
	}
	return nil
}
