package serving_endpoints

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsEndpointID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"31ee674b7e6643f6930d5e12148fbe6f", true},
		{"31EE674B7E6643F6930D5E12148FBE6F", true},
		{"root", true},
		{"31ee674b-7e66-43f6-930d-5e12148fbe6f", false},
		{"31ee674b7e6643f6930d5e12148fbe6", false},
		{"my-endpoint", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isEndpointID(tt.in))
		})
	}
}

func TestResolveEndpointID(t *testing.T) {
	lookupErr := errors.New("Endpoint with name 'no-such-endpoint' does not exist.")
	tests := []struct {
		name     string
		arg      string
		endpoint *serving.ServingEndpointDetailed
		err      error
		want     string
		wantErr  string
	}{
		{name: "an ID is passed through", arg: "31ee674b7e6643f6930d5e12148fbe6f", want: "31ee674b7e6643f6930d5e12148fbe6f"},
		{name: "root is passed through", arg: "root", want: "root"},
		{name: "a name resolves to its ID", arg: "my-endpoint", endpoint: &serving.ServingEndpointDetailed{Id: "31ee674b7e6643f6930d5e12148fbe6f"}, want: "31ee674b7e6643f6930d5e12148fbe6f"},
		{name: "a failed lookup returns the lookup error", arg: "no-such-endpoint", err: lookupErr, wantErr: `"no-such-endpoint" is not a serving endpoint ID, and looking it up as an endpoint name failed: Endpoint with name 'no-such-endpoint' does not exist.`},
		{name: "an endpoint without an ID is an error", arg: "databricks-gpt-5-4", endpoint: &serving.ServingEndpointDetailed{Name: "databricks-gpt-5-4"}, wantErr: `serving endpoint "databricks-gpt-5-4" has no endpoint ID`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewMockWorkspaceClient(t)
			if tt.endpoint != nil || tt.err != nil {
				m.GetMockServingEndpointsAPI().EXPECT().GetByName(mock.Anything, tt.arg).Return(tt.endpoint, tt.err)
			}
			ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)

			got, err := resolveEndpointID(ctx, tt.arg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
