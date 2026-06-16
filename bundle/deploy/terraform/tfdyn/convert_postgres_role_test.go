package tfdyn

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPostgresRole(t *testing.T) {
	src := resources.PostgresRole{
		PostgresRoleConfig: resources.PostgresRoleConfig{
			RoleId: "my-role",
			Parent: "projects/my-project/branches/main",
			RoleRoleSpec: postgres.RoleRoleSpec{
				PostgresRole: "my_postgres_role",
				IdentityType: postgres.RoleIdentityTypeUser,
				AuthMethod:   postgres.RoleAuthMethodLakebaseOauthV1,
				Attributes: &postgres.RoleAttributes{
					Createdb: true,
				},
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresRoleConverter{}.Convert(ctx, "my_postgres_role", vin, out)
	require.NoError(t, err)

	postgresRole := out.PostgresRole["my_postgres_role"]
	assert.Equal(t, map[string]any{
		"role_id": "my-role",
		"parent":  "projects/my-project/branches/main",
		"spec": map[string]any{
			"postgres_role": "my_postgres_role",
			"identity_type": "USER",
			"auth_method":   "LAKEBASE_OAUTH_V1",
			"attributes": map[string]any{
				"createdb": true,
			},
		},
	}, postgresRole)
}

func TestConvertPostgresRoleMinimal(t *testing.T) {
	src := resources.PostgresRole{
		PostgresRoleConfig: resources.PostgresRoleConfig{
			RoleId: "minimal-role",
			Parent: "projects/my-project/branches/main",
			RoleRoleSpec: postgres.RoleRoleSpec{
				PostgresRole: "minimal_role",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresRoleConverter{}.Convert(ctx, "minimal_postgres_role", vin, out)
	require.NoError(t, err)

	postgresRole := out.PostgresRole["minimal_postgres_role"]
	assert.Equal(t, map[string]any{
		"role_id": "minimal-role",
		"parent":  "projects/my-project/branches/main",
		"spec": map[string]any{
			"postgres_role": "minimal_role",
		},
	}, postgresRole)
}
