package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.PostgresDatabase{
			DatabaseId: "my-postgres-database",
			Parent:     "projects/my-postgres-project/branches/my-postgres-branch",
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockPostgresAPI().EXPECT().GetDatabase(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
