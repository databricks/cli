package yamlsaver

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestConvertToMapValueWithOrder(t *testing.T) {
	type test struct {
		Name            string            `json:"name"`
		Map             map[string]string `json:"map"`
		List            []string          `json:"list"`
		LongNameField   string            `json:"long_name_field"`
		ForceSendFields []string          `json:"-"`
		Format          string            `json:"format"`
	}

	v := &test{
		Name: "test",
		Map: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		List: []string{"a", "b", "c"},
		ForceSendFields: []string{
			"Name",
		},
		LongNameField: "long name goes here",
	}
	result, err := ConvertToMapValue(v, NewOrder([]string{"list", "name", "map"}), map[string]dyn.Value{})
	assert.NoError(t, err)

	assert.Equal(t, map[string]dyn.Value{
		"list": dyn.NewValue([]dyn.Value{
			dyn.V("a"),
			dyn.V("b"),
			dyn.V("c"),
		}, dyn.Location{Line: -3}),
		"name": dyn.NewValue("test", dyn.Location{Line: -2}),
		"map": dyn.NewValue(map[string]dyn.Value{
			"key1": dyn.V("value1"),
			"key2": dyn.V("value2"),
		}, dyn.Location{Line: -1}),
		"long_name_field": dyn.NewValue("long name goes here", dyn.Location{Line: 1}),
	}, result.MustMap())
}
