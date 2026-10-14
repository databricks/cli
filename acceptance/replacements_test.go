package acceptance_test

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentIDReplacement(t *testing.T) {
	var config internal.TestConfig
	_, err := toml.DecodeFile("test.toml", &config)
	require.NoError(t, err)
	repls := testdiff.ReplacementsContext{Repls: config.Repls}

	for _, tc := range []struct {
		name string
		old  string
		new  string
	}{
		{
			name: "short ID",
			old:  `/api/2.0/bundle/deployments/12/versions/2/operations/jobs.foo`,
			new:  `/api/2.0/bundle/deployments/[DEPLOYMENT_ID]/versions/2/operations/jobs.foo`,
		},
		{
			name: "long ID",
			old:  `"name": "deployments/8900000000000000001"`,
			new:  `"name": "deployments/[DEPLOYMENT_ID]"`,
		},
		{
			name: "versions and counts",
			old:  `{"version_id":"12","sequence_id":12,"count":12}`,
			new:  `{"version_id":"12","sequence_id":12,"count":12}`,
		},
		{
			name: "nonnumeric ID",
			old:  `deployments/12abc`,
			new:  `deployments/12abc`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.new, repls.Replace(tc.old))
		})
	}
}

func TestWorkspaceListReplacement(t *testing.T) {
	var config internal.TestConfig
	_, err := toml.DecodeFile("bundle/migrate/test.toml", &config)
	require.NoError(t, err)
	repls := testdiff.ReplacementsContext{Repls: config.Repls}

	for _, tc := range []struct {
		name string
		old  string
		new  string
	}{
		{
			name: "four-digit IDs",
			old:  "ID    Type  Language  Path\n",
			new:  "ID                   Type  Language  Path\n",
		},
		{
			name: "five-digit IDs",
			old:  "ID     Type  Language  Path\n",
			new:  "ID                   Type  Language  Path\n",
		},
		{
			name: "mixed ID widths",
			old:  "9999   FILE            /state/old.json\n10000  FILE            /state/new.json\n",
			new:  "[OBJECT_ID]  FILE            /state/old.json\n[OBJECT_ID]  FILE            /state/new.json\n",
		},
		{
			name: "other table",
			old:  "ID    Name\n",
			new:  "ID    Name\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.new, repls.Replace(tc.old))
		})
	}
}
