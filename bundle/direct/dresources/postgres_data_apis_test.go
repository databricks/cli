package dresources

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
)

func init() {
	registerTestDeps("postgres_data_apis", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		// Creating the project implicitly provisions the "production" branch and
		// its primary read-write endpoint, which the Data API requires.
		_, err := client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
			ProjectId: "test-project-for-data-api",
			Project: postgres.Project{
				Spec: &postgres.ProjectSpec{
					DisplayName: "Test Project for Data API",
					PgVersion:   16,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		waiter, err := client.Postgres.CreateDatabase(ctx, postgres.CreateDatabaseRequest{
			Parent:     "projects/test-project-for-data-api/branches/production",
			DatabaseId: "test-database",
			Database: postgres.Database{
				Spec: &postgres.DatabaseDatabaseSpec{
					PostgresDatabase: "app_db",
					Role:             "projects/test-project-for-data-api/branches/production/roles/owner",
				},
			},
		})
		if err != nil {
			return nil, err
		}
		database, err := waiter.Wait(ctx)
		if err != nil {
			return nil, err
		}

		return &resources.PostgresDataApi{
			Parent:    database.Name,
			DbSchemas: []string{"public"},
		}, nil
	})

	registerCreateRecoveryOptOut("postgres_data_apis", "candidate: a database has one Data API, but the read response takes its spec from the status, which can be empty while it is being enabled, so a recovered Data API needs a wait the create operation would otherwise do")
}

func TestDataApiSpecFromStatus(t *testing.T) {
	jwtCache := duration.New(5 * time.Minute)

	tests := []struct {
		name   string
		status *postgres.DataApiDataApiStatus
		want   postgres.DataApiDataApiSpec
	}{
		{
			name:   "no status",
			status: nil,
			want:   postgres.DataApiDataApiSpec{},
		},
		{
			name: "every spec field is read back; url and available_schemas are not",
			status: &postgres.DataApiDataApiStatus{
				AvailableSchemas:         []string{"public", "api"},
				DbAggregatesEnabled:      true,
				DbExtraSearchPath:        []string{"extensions"},
				DbMaxRows:                100,
				DbSchemas:                []string{"api"},
				JwtCacheMaxLifetime:      jwtCache,
				JwtRoleClaimKey:          ".sub",
				OpenapiMode:              postgres.OpenApiModeOpenApiModeIgnorePrivileges,
				ServerCorsAllowedOrigins: []string{"https://app.example.com"},
				ServerTimingEnabled:      true,
				Url:                      "https://ep.example.com/rest/app_db",
			},
			want: postgres.DataApiDataApiSpec{
				DbAggregatesEnabled:      true,
				DbExtraSearchPath:        []string{"extensions"},
				DbMaxRows:                100,
				DbSchemas:                []string{"api"},
				JwtCacheMaxLifetime:      jwtCache,
				JwtRoleClaimKey:          ".sub",
				OpenapiMode:              postgres.OpenApiModeOpenApiModeIgnorePrivileges,
				ServerCorsAllowedOrigins: []string{"https://app.example.com"},
				ServerTimingEnabled:      true,
				ForceSendFields:          []string{"DbAggregatesEnabled", "ServerTimingEnabled"},
			},
		},
		{
			// Status omits a false bool that was never set; it still means false.
			name:   "omitted bools are false",
			status: &postgres.DataApiDataApiStatus{DbSchemas: []string{"public"}},
			want: postgres.DataApiDataApiSpec{
				DbSchemas:       []string{"public"},
				ForceSendFields: []string{"DbAggregatesEnabled", "ServerTimingEnabled"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dataApiSpecFromStatus(tt.status))
		})
	}
}
