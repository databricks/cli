package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("external_locations", &resources.ExternalLocation{
		Name:           "myexternallocation",
		Url:            "s3://mybucket/mypath",
		CredentialName: "mycredential",
		Comment:        "Test external location",
	})

	registerTestDeps("external_locations.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "external_location",
			FullName:      "myexternallocation",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeReadFiles},
				Principal:  "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInRemoteType("external_locations",
		"skip_validation",
	)
}
