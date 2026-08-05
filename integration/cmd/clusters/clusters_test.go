package clusters_test

import (
	"regexp"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestClustersList(t *testing.T) {
	ctx := t.Context()
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "clusters", "list")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Name")
	assert.Contains(t, outStr, "State")
	assert.Empty(t, stderr.String())

	idRegExp := regexp.MustCompile(`[0-9]{4}\-[0-9]{6}-[a-z0-9]{8}`)
	clusterId := idRegExp.FindString(outStr)
	assert.NotEmpty(t, clusterId)
}

func TestClusterCreateErrorWhenNoArguments(t *testing.T) {
	ctx := t.Context()
	_, _, err := testcli.RequireErrorRun(t, ctx, "clusters", "create")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}
