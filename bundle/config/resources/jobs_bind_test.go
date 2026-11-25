package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.Job{
			JobSettings: jobs.JobSettings{},
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockJobsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
