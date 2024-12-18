package pythontest

import (
	"context"
	"testing"
)

func TestVenv(t *testing.T) {
	// Test at least two version to ensure we capture a case where venv version does not match system one
	for _, pythonVersion := range []string{"3.11", "3.12"} {
		t.Run(pythonVersion, func(t *testing.T) {
			ctx := context.Background()
			RequirePythonVENV(t, ctx, pythonVersion, true)
		})
	}
}
