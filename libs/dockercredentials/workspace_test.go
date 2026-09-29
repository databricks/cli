package dockercredentials_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	mockcatalog "github.com/databricks/databricks-sdk-go/experimental/mocks/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceID(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		got, err := dockercredentials.WorkspaceID(t.Context(), profile.Profile{WorkspaceID: "123"}, nil)
		require.NoError(t, err)
		assert.Equal(t, "123", got)
	})

	for _, id := range []string{"", auth.WorkspaceIDNone} {
		t.Run(id, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
				assert.Empty(t, req.Headers.Get(auth.WorkspaceIDHeader))
				return testserver.Response{Headers: http.Header{"X-Databricks-Org-Id": {"456"}}, Body: map[string]any{}}
			})
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
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
			metastores := mockcatalog.NewMockMetastoresInterface(t)
			metastores.EXPECT().Summary(mock.Anything).Return(&catalog.GetMetastoreSummaryResponse{Region: tt.region}, tt.err)
			got, err := dockercredentials.WorkspaceRegion(t.Context(), profile.Profile{Name: "workspace"}, metastores)
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
