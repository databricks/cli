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

func TestNewGenerateExternalLocationCommand_Help(t *testing.T) {
	cmd := NewGenerateExternalLocationCommand()
	assert.Contains(t, cmd.Long, "external location")
	assert.NotNil(t, cmd.Flag("existing-external-location-name"))
}

func TestGenerateExternalLocation_WritesYAML(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockExternalLocationsAPI().EXPECT().
		GetByName(mock.Anything, "prod_loc").
		Return(&catalog.ExternalLocationInfo{
			Name:           "prod_loc",
			Url:            "s3://prod-bucket/loc",
			CredentialName: "prod_cred",
			Comment:        "prod location",
			ReadOnly:       true,
		}, nil)

	_, err := runSubcmd(t, w,
		"external-location",
		"--existing-external-location-name", "prod_loc",
		"--output-dir", work,
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "external_locations_prod_loc.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "external_locations:")
	assert.Contains(t, contents, "url: s3://prod-bucket/loc")
	assert.Contains(t, contents, "credential_name: prod_cred")
	assert.Contains(t, contents, "read_only: true")
}
