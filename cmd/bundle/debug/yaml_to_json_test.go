package debug

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlToJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.yml")
	require.NoError(t, os.WriteFile(path, []byte("name: example\nvalues: [1, true, null]\n"), 0o644))

	var out bytes.Buffer
	require.NoError(t, yamlToJSON(path, &out))

	assert.JSONEq(t, `{"name":"example","values":[1,true,null]}`, out.String())
}

func TestYamlToJSONMissingFile(t *testing.T) {
	err := yamlToJSON(filepath.Join(t.TempDir(), "missing.yml"), &bytes.Buffer{})

	assert.ErrorIs(t, err, os.ErrNotExist)
}
