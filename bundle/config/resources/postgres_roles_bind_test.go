package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.PostgresRole{
			RoleId:       "my-postgres-role",
			Parent:       "projects/my-postgres-project/branches/my-postgres-branch",
			PostgresRole: "my_postgres_role",
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockPostgresAPI().EXPECT().GetRole(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
