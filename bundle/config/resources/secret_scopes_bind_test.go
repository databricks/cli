package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.SecretScope{
			Name: "0",
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockSecretsAPI().EXPECT().ListScopesAll(mock.Anything).Return([]workspace.SecretScope{
				{Name: "0"},
			}, nil)
		},
	})
}
