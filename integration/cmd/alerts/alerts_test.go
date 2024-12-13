package alerts_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestAlertsCreateErrWhenNoArguments(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "alerts-legacy", "create")
	assert.Equal(t, "please provide command input in JSON format by specifying the --json flag", err.Error())
}
