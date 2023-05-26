package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/databricks/cli/cmd/workspace"
)

func TestSecretsCreateScopeErrWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "secrets", "create-scope")
	assert.Equal(t, "accepts 1 arg(s), received 0", err.Error())
}
