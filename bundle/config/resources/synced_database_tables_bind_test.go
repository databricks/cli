package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.SyncedDatabaseTable{
			SyncedDatabaseTable: database.SyncedDatabaseTable{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockDatabaseAPI().EXPECT().GetSyncedDatabaseTable(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
