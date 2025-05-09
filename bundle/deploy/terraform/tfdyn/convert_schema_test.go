package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSchema(t *testing.T) {
	src := resources.Schema{
		CreateSchema: catalog.CreateSchema{
			Name:        "name",
			CatalogName: "catalog",
			Comment:     "comment",
			Properties: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			StorageRoot: "root",
		},
		Grants: []resources.Grant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jack@gmail.com",
			},
			{
				Privileges: []string{"RUN"},
				Principal:  "jane@gmail.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = schemaConverter{}.Convert(ctx, "my_schema", vin, out)
	require.NoError(t, err)

	// Assert equality on the schema
	assert.Equal(t, map[string]any{
		"name":         "name",
		"catalog_name": "catalog",
		"comment":      "comment",
		"properties": map[string]any{
			"k1": "v1",
			"k2": "v2",
		},
		"force_destroy": true,
		"storage_root":  "root",
	}, out.Schema["my_schema"])

	// Assert equality on the grants
	assert.Equal(t, &schema.ResourceGrants{
		Schema: "${databricks_schema.my_schema.id}",
		Grant: []schema.ResourceGrantsGrant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jack@gmail.com",
			},
			{
				Privileges: []string{"RUN"},
				Principal:  "jane@gmail.com",
			},
		},
	}, out.Grants["schema_my_schema"])
}
