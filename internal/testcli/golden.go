package testcli

import (
	"context"
	"fmt"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/assert"
)

func captureOutput(t testutil.TestingT, ctx context.Context, args []string) string {
	t.Helper()
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	assert.NoError(t, err)
	return stderr.String() + stdout.String()
}

func AssertOutput(t testutil.TestingT, ctx context.Context, args []string, expectedPath string) {
	t.Helper()
	out := captureOutput(t, ctx, args)
	testdiff.AssertOutput(t, ctx, out, fmt.Sprintf("Output from %v", args), expectedPath)
}
