package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.InstancePool{},
		id:       "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockInstancePoolsAPI().EXPECT().GetByInstancePoolId(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
