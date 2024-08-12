package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/stretchr/testify/assert"
)

func TestUndefinedJobLoadsWithError(t *testing.T) {
	b := load(t, "./undefined_job")
	diags := bundle.Apply(context.Background(), b, validate.AllResourcesHaveValues())
	assert.ErrorContains(t, diags.Error(), "job undefined is not defined")
}
