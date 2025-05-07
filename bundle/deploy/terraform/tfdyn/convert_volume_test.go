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

func TestConvertVolume(t *testing.T) {
	src := resources.Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName:     "catalog",
			Comment:         "comment",
			Name:            "name",
			SchemaName:      "schema",
			StorageLocation: "s3://bucket/path",
			VolumeType:      "EXTERNAL",
		},
		Grants: []resources.Grant{
			{
				Privileges: []string{"READ_VOLUME"},
				Principal:  "jack@gmail.com",
			},
			{
				Privileges: []string{"WRITE_VOLUME"},
				Principal:  "jane@gmail.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = volumeConverter{}.Convert(ctx, "my_volume", vin, out)
	require.NoError(t, err)

	// Assert equality on the volume
	require.Equal(t, map[string]any{
		"catalog_name":     "catalog",
		"comment":          "comment",
		"name":             "name",
		"schema_name":      "schema",
		"storage_location": "s3://bucket/path",
		"volume_type":      "EXTERNAL",
	}, out.Volume["my_volume"])

	// Assert equality on the grants
	assert.Equal(t, &schema.ResourceGrants{
		Volume: "${databricks_volume.my_volume.id}",
		Grant: []schema.ResourceGrantsGrant{
			{
				Privileges: []string{"READ_VOLUME"},
				Principal:  "jack@gmail.com",
			},
			{
				Privileges: []string{"WRITE_VOLUME"},
				Principal:  "jane@gmail.com",
			},
		},
	}, out.Grants["volume_my_volume"])
}
