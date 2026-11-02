package structvar_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestYAMLErrorMapMerge(t *testing.T) {
	for _, file := range []string{
		"testdata/yaml/error_01.yml",
		"testdata/yaml/error_02.yml",
		"testdata/yaml/error_03.yml",
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
				_, _, err := decodeFile(t, file)
				assert.ErrorContains(t, err, "map merge requires map or sequence of maps as the value")
			})
		})
	}
}
