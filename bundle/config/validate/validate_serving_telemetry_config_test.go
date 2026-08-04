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

func telemetryConfig() *serving.TelemetryConfig {
	return &serving.TelemetryConfig{
		InferenceTableConfig: &serving.TelemetryInferenceTableConfig{SamplingFraction: 0.5},
	}
}

func TestValidateServingTelemetryConfig(t *testing.T) {
	tests := []struct {
		name     string
		endpoint serving.CreateServingEndpoint
		wantErr  bool
	}{
		{
			name: "telemetry with a served entity",
			endpoint: serving.CreateServingEndpoint{
				Name:            "my_endpoint",
				TelemetryConfig: telemetryConfig(),
				Config: &serving.EndpointCoreConfigInput{
					ServedEntities: []serving.ServedEntityInput{{Name: "prod"}},
				},
			},
		},
		{
			// served_models is converted to served_entities by ModelServingEndpointFixups,
			// so accept either to stay independent of mutator order.
			name: "telemetry with a served model",
			endpoint: serving.CreateServingEndpoint{
				Name:            "my_endpoint",
				TelemetryConfig: telemetryConfig(),
				Config: &serving.EndpointCoreConfigInput{
					ServedModels: []serving.ServedModelInput{{ModelName: "m", ModelVersion: "1"}},
				},
			},
		},
		{
			name: "no telemetry and nothing served",
			endpoint: serving.CreateServingEndpoint{
				Name: "my_endpoint",
			},
		},
		{
			name: "telemetry and no config",
			endpoint: serving.CreateServingEndpoint{
				Name:            "my_endpoint",
				TelemetryConfig: telemetryConfig(),
			},
			wantErr: true,
		},
		{
			name: "telemetry and a config that serves nothing",
			endpoint: serving.CreateServingEndpoint{
				Name:            "my_endpoint",
				TelemetryConfig: telemetryConfig(),
				Config:          &serving.EndpointCoreConfigInput{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := ValidateServingTelemetryConfig().Apply(t.Context(), endpointBundle(tt.endpoint))

			if !tt.wantErr {
				assert.Empty(t, diags)
				return
			}

			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, "telemetry_config is not supported on an endpoint that does not serve a model", diags[0].Summary)
			assert.Equal(t, "resources.model_serving_endpoints.my_endpoint.telemetry_config", diags[0].Paths[0].String())
		})
	}
}
