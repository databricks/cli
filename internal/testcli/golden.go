package testcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/assert"
)

func captureOutput(t testutil.TestingT, ctx context.Context, args []string) string {
	t.Logf("run args: [%s]", strings.Join(args, ", "))
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	assert.NoError(t, err)
	return stderr.String() + stdout.String()
}

func AssertOutput(t testutil.TestingT, ctx context.Context, args []string, expectedPath string) {
	out := captureOutput(t, ctx, args)
	testdiff.AssertOutput(t, ctx, out, fmt.Sprintf("Output from %v", args), expectedPath)
}

func AssertOutputJQ(t testutil.TestingT, ctx context.Context, args []string, expectedPath string, ignorePaths []string) {
	out := captureOutput(t, ctx, args)
	testdiff.AssertOutputJQ(t, ctx, fmt.Sprintf("Output from %v", args), out, expectedPath, ignorePaths)
}
