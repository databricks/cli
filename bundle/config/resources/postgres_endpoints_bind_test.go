package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.PostgresEndpoint{
			EndpointId:   "my-postgres-endpoint",
			Parent:       "projects/my-postgres-project/branches/my-postgres-branch",
			EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockPostgresAPI().EXPECT().GetEndpoint(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
