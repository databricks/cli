//go:build !wasm

package main

import (
	"testing"

	"github.com/databricks/cli/libs/template"
	"github.com/stretchr/testify/require"
)

func TestRenderEmpty(t *testing.T) {
	out, err := template.Render("default-python", nil, nil)
	require.NoError(t, err)
	require.Contains(t, out, "my_project/databricks.yml")

	main := out["my_project/databricks.yml"]
	require.Contains(t, main, "my_project")
}

func TestRenderUserName(t *testing.T) {
	out, err := template.Render("default-python", nil, map[string]string{"user_name": "another_user"})
	require.NoError(t, err)
	require.Contains(t, out, "my_project/databricks.yml")

	main := out["my_project/databricks.yml"]
	require.Contains(t, main, "user_name: another_user")
}
