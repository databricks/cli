package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateVolumeCommand_Help(t *testing.T) {
	cmd := NewGenerateVolumeCommand()
	assert.Contains(t, cmd.Long, "Unity Catalog volume")
	assert.NotNil(t, cmd.Flag("existing-volume-name"))
}

func TestGenerateVolume_External_PreservesStorageLocation(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockVolumesAPI().EXPECT().
		ReadByName(mock.Anything, "prod.raw.landing").
		Return(&catalog.VolumeInfo{
			Name:            "landing",
			CatalogName:     "prod",
			SchemaName:      "raw",
			VolumeType:      catalog.VolumeTypeExternal,
			StorageLocation: "s3://bucket/landing",
			Comment:         "raw landings",
		}, nil)

	_, err := runSubcmd(t, w,
		"volume",
		"--existing-volume-name", "prod.raw.landing",
		"--output-dir", work,
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "volumes_prod_raw_landing.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "volumes:")
	assert.Contains(t, contents, "storage_location: s3://bucket/landing")
	assert.Contains(t, contents, "volume_type: EXTERNAL")
}

func TestGenerateVolume_Managed_DropsStorageLocation(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockVolumesAPI().EXPECT().
		ReadByName(mock.Anything, "prod.raw.managed").
		Return(&catalog.VolumeInfo{
			Name:            "managed",
			CatalogName:     "prod",
			SchemaName:      "raw",
			VolumeType:      catalog.VolumeTypeManaged,
			StorageLocation: "s3://server-assigned/whatever",
		}, nil)

	_, err := runSubcmd(t, w,
		"volume",
		"--existing-volume-name", "prod.raw.managed",
		"--output-dir", work,
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "volumes_prod_raw_managed.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.NotContains(t, contents, "storage_location")
	assert.Contains(t, contents, "volume_type: MANAGED")
}
