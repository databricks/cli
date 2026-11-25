package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.Secret{
			CatalogName: "main",
			SchemaName:  "default",
			Name:        "my_secret",
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockSecretsUcAPI().EXPECT().GetSecret(mock.Anything, mock.Anything).Return(&catalog.Secret{FullName: "0"}, nil)
		},
	})
}
