package config

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/safeerr"
	"github.com/stretchr/testify/assert"
)

func TestResourceKeySafeString(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{key: "resources.jobs.my_job", want: "jobs.*"},
		{key: "resources.pipelines.my_pipeline", want: "pipelines.*"},
		{key: "resources.jobs.my_job.permissions", want: "jobs.*.permissions"},
		{key: "resources.schemas.my_schema.grants", want: "schemas.*.grants"},
		{key: "resources.secret_scopes.my scope.permissions", want: "secret_scopes.*.permissions"},

		// Shapes GetResourceTypeFromKey does not recognize report nothing.
		{key: "resources.jobs", want: "*"},
		{key: "jobs.my_job", want: "*"},
		{key: "", want: "*"},
		{key: "/Workspace/Users/someone@example.com/x", want: "*"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, ResourceKey(tt.key).SafeString())
		})
	}
}

// TestResourceKeyFormatsAsTheFullKey is what keeps error messages unchanged
// when a call site starts passing ResourceKey instead of a bare string.
func TestResourceKeyFormatsAsTheFullKey(t *testing.T) {
	const key = "resources.jobs.my_job"

	for _, format := range []string{"%s", "%q", "%v"} {
		t.Run(format, func(t *testing.T) {
			assert.Equal(t,
				fmt.Sprintf(format, key),
				fmt.Sprintf(format, ResourceKey(key)))
		})
	}
}

func TestResourceKeyInSafeerr(t *testing.T) {
	err := safeerr.Errorf("%s: SaveState: %w",
		ResourceKey("resources.jobs.my_job"), safeerr.New("disk full"))

	assert.Equal(t, "resources.jobs.my_job: SaveState: disk full", err.Error())
	assert.Equal(t, "jobs.*: SaveState: disk full", safeerr.ErrorTemplate(err))
	assert.NotContains(t, safeerr.ErrorTemplate(err), "my_job")
}
