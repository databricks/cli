package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.PostgresSnapshotSchedule{
			Branch: "projects/my-postgres-project/branches/my-postgres-branch",
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockPostgresAPI().EXPECT().GetSnapshotSchedule(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
