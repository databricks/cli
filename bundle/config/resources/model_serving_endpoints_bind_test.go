package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.ModelServingEndpoint{
			CreateServingEndpoint: serving.CreateServingEndpoint{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockServingEndpointsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
