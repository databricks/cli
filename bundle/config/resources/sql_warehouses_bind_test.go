package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.SqlWarehouse{
			CreateWarehouseRequest: sql.CreateWarehouseRequest{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockWarehousesAPI().EXPECT().GetById(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
