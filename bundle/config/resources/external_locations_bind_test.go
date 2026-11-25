package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.ExternalLocation{
			CreateExternalLocation: catalog.CreateExternalLocation{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockExternalLocationsAPI().EXPECT().GetByName(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
