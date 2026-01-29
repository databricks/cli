package legacytemplates_test

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/cmd/apps/legacytemplates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFuncsUCVolumeBindings(t *testing.T) {
	// Create resource values with a UC volume
	resources := legacytemplates.NewResourceValues()
	resources.Set(legacytemplates.ResourceTypeUCVolume, "main.default.my_volume")

	// Create a simple template that uses UC volume bindings
	tmplText := `
{{- if hasResource "uc_volume"}}
volume_present: true
{{- if hasBindings "uc_volume"}}
has_bindings: true
{{- range getBindingLines "uc_volume"}}
{{.}}
{{- end}}
{{- end}}
{{- end}}`

	tmpl, err := template.New("test").Funcs(legacytemplates.GetTemplateFuncsForTest(resources, nil)).Parse(tmplText)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "volume_present: true")
	assert.Contains(t, output, "has_bindings: true")
	assert.Contains(t, output, "volume:")
	assert.Contains(t, output, "name: ${var.uc_volume}")
	assert.Contains(t, output, "permission: READ_WRITE")
}

func TestTemplateFuncsSQLWarehouseBindings(t *testing.T) {
	// Create resource values with a SQL warehouse
	resources := legacytemplates.NewResourceValues()
	resources.Set(legacytemplates.ResourceTypeSQLWarehouse, "abc123")

	// Create a simple template that uses SQL warehouse bindings
	tmplText := `
{{- if hasResource "sql_warehouse"}}
warehouse_present: true
{{- if hasBindings "sql_warehouse"}}
has_bindings: true
{{- range getBindingLines "sql_warehouse"}}
{{.}}
{{- end}}
{{- end}}
{{- end}}`

	tmpl, err := template.New("test").Funcs(legacytemplates.GetTemplateFuncsForTest(resources, nil)).Parse(tmplText)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "warehouse_present: true")
	assert.Contains(t, output, "has_bindings: true")
	assert.Contains(t, output, "sql_warehouse:")
	assert.Contains(t, output, "id: ${var.warehouse_id}")
	assert.Contains(t, output, "permission: CAN_USE")
}

func TestTemplateFuncsMultipleResources(t *testing.T) {
	// Create resource values with multiple resources including UC volume
	resources := legacytemplates.NewResourceValues()
	resources.Set(legacytemplates.ResourceTypeUCVolume, "main.default.my_volume")
	resources.Set(legacytemplates.ResourceTypeSQLWarehouse, "warehouse123")

	// Verify both resources are present and have bindings
	tmplText := `
{{- range resourceTypes}}
{{- if hasResource .}}
- {{.}}: {{hasBindings .}}
{{- end}}
{{- end}}`

	tmpl, err := template.New("test").Funcs(legacytemplates.GetTemplateFuncsForTest(resources, nil)).Parse(tmplText)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "sql_warehouse: true")
	assert.Contains(t, output, "uc_volume: true")
}

func TestTemplateFuncsNoBindingsForMissingResource(t *testing.T) {
	// Empty resource values
	resources := legacytemplates.NewResourceValues()

	tmplText := `
{{- if hasResource "uc_volume"}}
volume_present: true
{{- else}}
volume_absent: true
{{- end}}`

	tmpl, err := template.New("test").Funcs(legacytemplates.GetTemplateFuncsForTest(resources, nil)).Parse(tmplText)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "volume_absent: true")
	assert.NotContains(t, output, "volume_present: true")
}

func TestGetVariableName(t *testing.T) {
	resources := legacytemplates.NewResourceValues()

	tmplText := `
{{- $varName := getVariableName "uc_volume"}}
{{- if $varName}}
variable_name: {{$varName}}
{{- end}}`

	tmpl, err := template.New("test").Funcs(legacytemplates.GetTemplateFuncsForTest(resources, nil)).Parse(tmplText)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, "variable_name: uc_volume", output)
}
