package dresources

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
)

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
