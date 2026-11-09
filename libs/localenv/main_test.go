package localenv

import (
	"os"
	"testing"
)

// TestMain clears uv's UV_PROJECT_ENVIRONMENT from the host shell: it moves the
// venv the pipeline uses, and most tests expect the default .venv. Tests that
// need it set it on the context.
func TestMain(m *testing.M) {
	os.Unsetenv(EnvUvProjectEnvironment) //nolint:usetesting // t.Setenv cannot unset
	m.Run()
}
