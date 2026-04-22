package terraform

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolate_RewritesUcmPathToTfPath(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"resource": dyn.V(map[string]dyn.Value{
			"databricks_catalog": dyn.V(map[string]dyn.Value{
				"sales": dyn.V(map[string]dyn.Value{
					"name":         dyn.V("sales_prod"),
					"storage_root": dyn.V("${resources.storage_credentials.sales_cred.name}"),
				}),
			}),
		}),
	})

	out, err := Interpolate(in)
	require.NoError(t, err)

	got, ok := out.Get("resource").Get("databricks_catalog").Get("sales").Get("storage_root").AsString()
	require.True(t, ok)
	assert.Equal(t, "${databricks_storage_credential.sales_cred.name}", got)
}

func TestInterpolate_LeavesLiteralsUntouched(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"resource": dyn.V(map[string]dyn.Value{
			"databricks_catalog": dyn.V(map[string]dyn.Value{
				"sales": dyn.V(map[string]dyn.Value{
					"storage_root": dyn.V("s3://acme-sales/prod"),
				}),
			}),
		}),
	})

	out, err := Interpolate(in)
	require.NoError(t, err)

	got, _ := out.Get("resource").Get("databricks_catalog").Get("sales").Get("storage_root").AsString()
	assert.Equal(t, "s3://acme-sales/prod", got)
}

func TestInterpolate_UnknownUcmKindPassesThrough(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"resource": dyn.V(map[string]dyn.Value{
			"databricks_catalog": dyn.V(map[string]dyn.Value{
				"sales": dyn.V(map[string]dyn.Value{
					"storage_root": dyn.V("${resources.unknown.foo.bar}"),
				}),
			}),
		}),
	})

	out, err := Interpolate(in)
	require.NoError(t, err)

	got, _ := out.Get("resource").Get("databricks_catalog").Get("sales").Get("storage_root").AsString()
	assert.Equal(t, "${resources.unknown.foo.bar}", got)
}
