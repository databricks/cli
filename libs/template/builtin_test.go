package template

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltin(t *testing.T) {
	out, err := builtin()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(out), 3)

	// Create a map of templates by name for easier lookup
	templates := make(map[string]*builtinTemplate)
	for _, tmpl := range out {
		templates[tmpl.Name] = &tmpl
	}

	// Verify all expected templates exist
	assert.Contains(t, templates, "dbt-sql")
	assert.Contains(t, templates, "default-python")
	assert.Contains(t, templates, "default-sql")

	// Verify the filesystems work for each template
	_, err = fs.Stat(templates["dbt-sql"].FS, `template/{{.project_name}}/dbt_project.yml.tmpl`)
	assert.NoError(t, err)
	_, err = fs.Stat(templates["default-python"].FS, `template/{{.project_name}}/tests/main_test.py.tmpl`)
	assert.NoError(t, err)
	_, err = fs.Stat(templates["default-sql"].FS, `template/{{.project_name}}/src/orders_daily.sql.tmpl`)
	assert.NoError(t, err)
}
