package root

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestInvokedByWithEnvironmentVariable(t *testing.T) {
	t.Setenv(invokedByEnvironmentVariable, "foobar")
	ctx := withInvokedByInUserAgent(context.Background())
	assert.Contains(t, useragent.FromContext(ctx), "invoked-by/foobar")
}

func TestInvokedByWithoutEnvironmentVariable(t *testing.T) {
	t.Setenv(invokedByEnvironmentVariable, "")
	os.Unsetenv(invokedByEnvironmentVariable)
	ctx := withInvokedByInUserAgent(context.Background())
	assert.NotContains(t, useragent.FromContext(ctx), "invoked-by/")
}
