package dockercredentials_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/options/call"
	"github.com/databricks/sdk-go/uc/metastores/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceID(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		got, err := dockercredentials.WorkspaceID(t.Context(), profile.Profile{WorkspaceID: "123"}, nil)
		require.NoError(t, err)
		assert.Equal(t, "123", got)
	})

	for _, id := range []string{"", profile.WorkspaceIDNone} {
		t.Run(id, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
				assert.Empty(t, req.Headers.Get(auth.WorkspaceIDHeader))
				return testserver.Response{Headers: http.Header{"X-Databricks-Org-Id": {"456"}}, Body: map[string]any{}}
			})
			w, err := sdkclients.New(t.Context(), &config.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)
			w.Config.WorkspaceID = "ambient-id"
			got, err := dockercredentials.WorkspaceID(t.Context(), profile.Profile{Name: "workspace", WorkspaceID: id}, w)
			require.NoError(t, err)
			assert.Equal(t, "456", got)
		})
	}
}

func TestWorkspaceRegion(t *testing.T) {
	wantErr := errors.New("summary failed")
	for _, tt := range []struct {
		name, region, want, wantError string
		err                           error
	}{
		{name: "trim", region: " us-west-2 ", want: "us-west-2"},
		{name: "missing", wantError: "metastore summary did not include a region"},
		{name: "blank", region: " \t", wantError: "metastore summary did not include a region"},
		{name: "error", err: wantErr, wantError: "summary failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := fakeSummarizer{resp: &metastores.GetMetastoreSummaryResponse{Region: new(tt.region)}, err: tt.err}
			got, err := dockercredentials.WorkspaceRegion(t.Context(), profile.Profile{Name: "workspace"}, api)
			if tt.wantError != "" {
				assert.ErrorContains(t, err, `resolve workspace region for profile "workspace": `+tt.wantError)
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

type fakeSummarizer struct {
	resp *metastores.GetMetastoreSummaryResponse
	err  error
}

func (f fakeSummarizer) GetMetastoreSummary(context.Context, metastores.GetMetastoreSummaryRequest, ...call.Option) (*metastores.GetMetastoreSummaryResponse, error) {
	return f.resp, f.err
}
