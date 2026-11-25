package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.Volume{
			CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockVolumesAPI().EXPECT().Read(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
