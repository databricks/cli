package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func TestMakeVolumeRemote_FromFullName(t *testing.T) {
	info := &catalog.VolumeInfo{
		CatalogName: "main",
		SchemaName:  "myschema",
		Name:        "myvolume",
		FullName:    "main.myschema.myvolume",
	}

	remote := makeVolumeRemote(info)

	assert.Equal(t, "/Volumes/main/myschema/myvolume", remote.VolumePath)
	assert.Equal(t, "main.myschema.myvolume", remote.FullName)
}

func TestMakeVolumeRemote_FallbackFromNameParts(t *testing.T) {
	info := &catalog.VolumeInfo{
		CatalogName: "main",
		SchemaName:  "myschema",
		Name:        "myvolume",
	}

	remote := makeVolumeRemote(info)

	assert.Equal(t, "/Volumes/main/myschema/myvolume", remote.VolumePath)
}

func TestResourceVolumeRemapState(t *testing.T) {
	r := &ResourceVolume{}

	state := r.RemapState(&VolumeRemote{
		VolumeInfo: catalog.VolumeInfo{
			CatalogName:     "main",
			SchemaName:      "myschema",
			Name:            "myvolume",
			Comment:         "comment",
			StorageLocation: "s3://bucket/path",
			VolumeType:      catalog.VolumeTypeManaged,
		},
		VolumePath: "/Volumes/main/myschema/myvolume",
	})

	assert.Equal(t, "main", state.CatalogName)
	assert.Equal(t, "myschema", state.SchemaName)
	assert.Equal(t, "myvolume", state.Name)
	assert.Equal(t, "comment", state.Comment)
	assert.Equal(t, "s3://bucket/path", state.StorageLocation)
	assert.Equal(t, catalog.VolumeTypeManaged, state.VolumeType)
}
