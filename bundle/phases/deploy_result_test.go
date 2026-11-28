package phases

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/env"
	libsenv "github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorClass(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"server error", &apierr.APIError{StatusCode: 500}, errorClassService},
		{"rate limited", &apierr.APIError{StatusCode: 429}, errorClassService},
		{"wrapped server error", fmt.Errorf("cannot create resources.jobs.foo: %w", &apierr.APIError{StatusCode: 503}), errorClassService},
		{"permission denied", &apierr.APIError{StatusCode: 403}, errorClassUser},
		{"no response", &url.Error{Op: "Post", URL: "https://host.test", Err: errors.New("no such host")}, errorClassUser},
		{"not an API error", errors.New("reference does not exist"), errorClassUser},
		{"no error to inspect", nil, errorClassUser},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, errorClass(c.err))
		})
	}
}

func TestWriteDeployResultOnExitReportsPanicAsService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	ctx := libsenv.Set(logdiag.InitContext(t.Context()), env.AutoDeployResultFileVariable, path)

	assert.PanicsWithValue(t, "boom", func() {
		var retErr error
		defer WriteDeployResultOnExit(ctx, &retErr)
		panic("boom")
	})

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"error_class": "SERVICE"}`, string(data))
}
