package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func TestFilterFields(t *testing.T) {
	fields := []string{"Comment", "NewName", "Owner", "Name", "NotExistingField"}
	result := filterFields[catalog.UpdateVolumeRequestContent](fields, "NewName", "Owner")
	expected := []string{"Comment", "Name"}
	assert.Equal(t, expected, result)
}
