package yamlloader_test

import (
	"bytes"
	"os"
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLErrorMapMerge(t *testing.T) {
	for _, file := range []string{
		"testdata/error_01.yml",
		"testdata/error_02.yml",
		"testdata/error_03.yml",
	} {
		input, err := os.ReadFile(file)
		require.NoError(t, err)

		t.Run(file, func(t *testing.T) {
			t.Run("reference", func(t *testing.T) {
				var ref any
				err = yaml.Unmarshal(input, &ref)
				assert.ErrorContains(t, err, "map merge requires map or sequence of maps as the value")
			})

			t.Run("self", func(t *testing.T) {
				_, err := yamlloader.LoadYAML(file, bytes.NewBuffer(input))
				assert.ErrorContains(t, err, "map merge requires map or sequence of maps as the value")
			})
		})
	}
}
