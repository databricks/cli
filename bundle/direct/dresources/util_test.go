package dresources

import (
	"testing"

	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
)

func TestFilterFields(t *testing.T) {
	type Foo struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
		D string `json:"d"`
	}
	fields := []string{"A", "B", "C", "NotExistingField"}
	result := utils.FilterFields[Foo](fields, "A", "D")
	expected := []string{"B", "C"}
	assert.Equal(t, expected, result)
}
