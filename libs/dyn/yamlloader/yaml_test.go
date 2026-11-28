package yamlloader_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func loadYAML(t *testing.T, path string) dyn.Value {
	input, err := os.ReadFile(path)
	require.NoError(t, err)

	var ref any
	err = yaml.Unmarshal(input, &ref)
	require.NoError(t, err)

	self, err := yamlloader.LoadYAML(path, bytes.NewBuffer(input))
	require.NoError(t, err)
	assert.NotNil(t, self)

	// Deep-equal the two values to ensure that the loader is producing
	assert.EqualValues(t, ref, self.AsAny())
	return self
}

func TestYAMLEmpty(t *testing.T) {
	self := loadYAML(t, "testdata/empty.yml")
	assert.Equal(t, dyn.NilValue, self)
}

func TestYAMLScalarSourceText(t *testing.T) {
	input := "int: 0042\nneg: -007\nfloat: 1.10\nalias: &a 0042\nref: *a\nbool: True\nquoted: \"0042\"\nstr: abc\nnull: ~\ndate: 2024-01-01\n"
	v, err := yamlloader.LoadYAML("file.yml", bytes.NewBufferString(input))
	require.NoError(t, err)

	for key, want := range map[string]string{"int": "0042", "neg": "-007", "float": "1.10", "alias": "0042", "ref": "0042"} {
		text, ok := v.Get(key).SourceText()
		assert.True(t, ok, key)
		assert.Equal(t, want, text, key)
	}

	for _, key := range []string{"bool", "quoted", "str", "null", "date"} {
		_, ok := v.Get(key).SourceText()
		assert.False(t, ok, key)
	}
}
