package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.GenieSpace{},
		id:       "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockGenieAPI().EXPECT().GetSpace(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
