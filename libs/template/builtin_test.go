package template

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltin(t *testing.T) {
	out, err := Builtin()
	require.NoError(t, err)
	assert.Len(t, out, 3)

	// Confirm names.
	assert.Equal(t, "dbt-sql", out[0].Name)
	assert.Equal(t, "default-python", out[1].Name)
	assert.Equal(t, "default-sql", out[2].Name)

	// Confirm that the filesystems work.
	_, err = fs.Stat(out[0].FS, `template/{{.project_name}}/dbt_project.yml.tmpl`)
	assert.NoError(t, err)
	_, err = fs.Stat(out[1].FS, `template/{{.project_name}}/tests/main_test.py.tmpl`)
	assert.NoError(t, err)
	_, err = fs.Stat(out[2].FS, `template/{{.project_name}}/src/orders_daily.sql.tmpl`)
	assert.NoError(t, err)
}
