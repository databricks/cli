package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectUnexpandedVar(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rawQuery string
		body     string
		want     string
		found    bool
	}{
		{
			name:  "braced uppercase var in body",
			path:  "/api/2.0/pipelines",
			body:  `{"name": "test-pipeline-${UNIQUE_NAME}"}`,
			want:  "${UNIQUE_NAME}",
			found: true,
		},
		{
			name:  "bare uppercase var in body",
			path:  "/api/2.0/pipelines",
			body:  `{"name": "test-pipeline-$UNIQUE_NAME"}`,
			want:  "$UNIQUE_NAME",
			found: true,
		},
		{
			name:  "uppercase var in path",
			path:  "/api/2.1/jobs/${JOB_ID}/get",
			found: true,
			want:  "${JOB_ID}",
		},
		{
			name:     "uppercase var in query",
			path:     "/api/2.1/jobs/get",
			rawQuery: "job_id=$JOB_ID",
			found:    true,
			want:     "$JOB_ID",
		},
		{
			name:  "DABs interpolation is ignored (lowercase, namespaced)",
			path:  "/api/2.1/unity-catalog/volumes",
			body:  `{"catalog_name": "${resources.volumes.bar.bad..syntax}"}`,
			found: false,
		},
		{
			name:  "lowercase bare var is ignored",
			path:  "/api/2.0/pipelines",
			body:  `{"x": "$job_id"}`,
			found: false,
		},
		{
			name:  "clean body",
			path:  "/api/2.0/pipelines",
			body:  `{"name": "test-pipeline-abc123"}`,
			found: false,
		},
		{
			name:  "dollar amounts are not variables",
			path:  "/api/2.0/pipelines",
			body:  `{"price": "$5", "positional": "$1", "literal": "$$"}`,
			found: false,
		},
		{
			name:  "file content endpoint is exempt",
			path:  "/api/2.0/workspace/import",
			body:  `{"content": "echo $PATH"}`,
			found: false,
		},
		{
			name:  "import-file endpoint is exempt",
			path:  "/api/2.0/workspace-files/import-file/Workspace/Users/x/notebook.py",
			body:  "print('${NOT_A_BUG}')",
			found: false,
		},
		{
			name:  "telemetry endpoint is exempt",
			path:  "/telemetry-ext",
			body:  `{"logs": ["name: test-${UNIQUE_NAME}"]}`,
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := detectUnexpandedVar(tt.path, tt.rawQuery, []byte(tt.body))
			assert.Equal(t, tt.found, found)
			if tt.found {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDetectUnexpandedVarAllowlist(t *testing.T) {
	allowedDollarTokens["${ALLOWED_EXAMPLE}"] = struct{}{}
	t.Cleanup(func() { delete(allowedDollarTokens, "${ALLOWED_EXAMPLE}") })

	_, found := detectUnexpandedVar("/api/2.0/pipelines", "", []byte(`{"x": "${ALLOWED_EXAMPLE}"}`))
	assert.False(t, found)
}
