package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.MlflowExperiment{
			CreateExperiment: ml.CreateExperiment{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockExperimentsAPI().EXPECT().GetExperiment(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
