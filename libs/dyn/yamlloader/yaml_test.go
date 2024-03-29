package yamlloader_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
