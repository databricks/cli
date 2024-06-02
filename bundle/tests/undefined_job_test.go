package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUndefinedJobLoadsWithError(t *testing.T) {
	_, diags := loadTargetWithDiags("./undefined_job", "default")
	assert.ErrorContains(t, diags.Error(), "job undefined is not defined")
}
