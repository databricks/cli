package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noProfile = "telemetry_config must name a table in table_names or set telemetry_profile_id"

func endpointBundle(endpoint serving.CreateServingEndpoint) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"my_endpoint": {CreateServingEndpoint: endpoint},
				},
			},
		},
	}
}

func servedModelConfig() *serving.EndpointCoreConfigInput {
	return &serving.EndpointCoreConfigInput{
		ServedEntities: []serving.ServedEntityInput{{
			Name:          "prod",
			EntityName:    "main.default.my_model",
			EntityVersion: "1",
		}},
	}
}

func TestValidateServingTelemetryConfig(t *testing.T) {
	tests := []struct {
		name     string
		endpoint serving.CreateServingEndpoint
		want     []string
	}{
		{
			name: "telemetry naming the tables to create a profile from",
			endpoint: serving.CreateServingEndpoint{
				Name: "my_endpoint",
				TelemetryConfig: &serving.TelemetryConfig{
					TableNames: &serving.UnityCatalogTableNames{LogsTable: "main.default.logs"},
				},
				Config: servedModelConfig(),
			},
		},
		{
			name: "telemetry pinned to an existing profile",
			endpoint: serving.CreateServingEndpoint{
				Name:            "my_endpoint",
				TelemetryConfig: &serving.TelemetryConfig{TelemetryProfileId: "abc"},
				Config:          servedModelConfig(),
			},
		},
		{
			name: "telemetry with only a traces table",
			endpoint: serving.CreateServingEndpoint{
				Name: "my_endpoint",
				TelemetryConfig: &serving.TelemetryConfig{
					TableNames: &serving.UnityCatalogTableNames{TracesTable: "main.default.traces"},
				},
				Config: servedModelConfig(),
			},
		},
		{
			name: "no telemetry at all",
			endpoint: serving.CreateServingEndpoint{
				Name:   "my_endpoint",
				Config: servedModelConfig(),
			},
		},
		{
			// Names no tables, so nothing identifies the profile.
			name: "telemetry with an empty table_names",
			endpoint: serving.CreateServingEndpoint{
				Name: "my_endpoint",
				TelemetryConfig: &serving.TelemetryConfig{
					TableNames: &serving.UnityCatalogTableNames{},
				},
				Config: servedModelConfig(),
			},
			want: []string{noProfile},
		},
		{
			// Discarded by the API, so the sampling fraction never takes effect.
			name: "telemetry with only an inference table config",
			endpoint: serving.CreateServingEndpoint{
				Name: "my_endpoint",
				TelemetryConfig: &serving.TelemetryConfig{
					InferenceTableConfig: &serving.TelemetryInferenceTableConfig{SamplingFraction: 0.5},
				},
				Config: servedModelConfig(),
			},
			want: []string{noProfile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := ValidateServingTelemetryConfig().Apply(t.Context(), endpointBundle(tt.endpoint))

			require.Len(t, diags, len(tt.want))
			for i, summary := range tt.want {
				assert.Equal(t, diag.Error, diags[i].Severity)
				assert.Equal(t, summary, diags[i].Summary)
				assert.Equal(t, "resources.model_serving_endpoints.my_endpoint.telemetry_config", diags[i].Paths[0].String())
			}
		})
	}
}
