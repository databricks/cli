package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.DataQualityMonitor{
			Monitor: dataquality.Monitor{},
		},
		// The ID has a structure that "0" does not satisfy.
		id: "table,0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockDataQualityAPI().EXPECT().GetMonitor(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
