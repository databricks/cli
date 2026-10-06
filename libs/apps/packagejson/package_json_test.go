package packagejson_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apps/packagejson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	for _, tt := range []struct {
		name   string
		prefix string
	}{
		{name: "JSON"},
		{name: "BOM", prefix: "\xef\xbb\xbf"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dir, packagejson.FileName), tt.prefix+`{
				"packageManager": "pnpm@10.30.3",
				"Scripts": {"build": "echo build"},
				"scripts": {"typecheck": null, "//": ["a comment"]},
				"largeNumber": 9007199254740993
			}`)
			fields, err := packagejson.Read(dir)
			require.NoError(t, err)
			assert.JSONEq(t, `"pnpm@10.30.3"`, string(fields["packageManager"]))
			assert.JSONEq(t, `{"build": "echo build"}`, string(fields["Scripts"]))
			assert.JSONEq(t, `{"typecheck": null, "//": ["a comment"]}`, string(fields["scripts"]))
			assert.Equal(t, "9007199254740993", string(fields["largeNumber"]))
		})
	}
}

func TestReadErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		fields, err := packagejson.Read(t.TempDir())
		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, fields)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		testutil.WriteFile(t, filepath.Join(dir, packagejson.FileName), "{")
		fields, err := packagejson.Read(dir)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*json.SyntaxError))
		assert.Nil(t, fields)
	})
}
