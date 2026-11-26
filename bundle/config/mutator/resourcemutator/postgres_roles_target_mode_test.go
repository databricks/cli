package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func init() {
	registerTargetModeExamples("postgres_roles", func(r *config.Resources) {
		r.PostgresRoles = map[string]*resources.PostgresRole{
			"postgres_role1": {
				PostgresRoleConfig: resources.PostgresRoleConfig{
					RoleId: "postgres-role-1",
					Parent: "projects/postgres-project-1/branches/postgres-branch-1",
					RoleRoleSpec: postgres.RoleRoleSpec{
						PostgresRole: "postgres_role_1",
					},
				},
			},
		}
	})
}
