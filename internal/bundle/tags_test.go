package bundle

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func testTags(t *testing.T, tags map[string]string) error {
	var nodeTypeId string
	switch testutil.GetCloud(t) {
	case testutil.AWS:
		nodeTypeId = "i3.xlarge"
	case testutil.Azure:
		nodeTypeId = "Standard_DS4_v2"
	case testutil.GCP:
		nodeTypeId = "n1-standard-4"
	}

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := w.Jobs.Create(ctx, jobs.CreateJob{
		Name: internal.RandomName("test-tags-"),
		Tasks: []jobs.Task{
			{
				TaskKey: "test",
				NewCluster: &compute.ClusterSpec{
					SparkVersion: "13.3.x-scala2.12",
					NumWorkers:   1,
					NodeTypeId:   nodeTypeId,
				},
				SparkPythonTask: &jobs.SparkPythonTask{
					PythonFile: "/doesnt_exist.py",
				},
			},
		},
		Tags: tags,
	})

	if resp != nil {
		t.Cleanup(func() {
			w.Jobs.DeleteByJobId(ctx, resp.JobId)
		})
	}

	return err
}

func testTagKey(t *testing.T, key string) error {
	return testTags(t, map[string]string{
		key: "value",
	})
}

func testTagValue(t *testing.T, value string) error {
	return testTags(t, map[string]string{
		"key": value,
	})
}

type tagTestCase struct {
	name  string
	value string
	fn    func(t *testing.T, value string) error
	err   string
}

func runTagTestCases(t *testing.T, cases []tagTestCase) {
	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.fn(t, tc.value)
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				msg := strings.ReplaceAll(err.Error(), "\n", " ")
				require.Contains(t, msg, tc.err)
			}
		})
	}
}

func TestAccTagKeyAWS(t *testing.T) {
	testutil.Require(t, testutil.AWS)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "invalid",
			value: "café",
			fn:    testTagKey,
			err:   ` The key must match the regular expression ^[\d \w\+\-=\.:\/@]*$.`,
		},
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagKey,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "empty",
			value: "",
			fn:    testTagKey,
			err:   ` the minimal length is 1, and the maximum length is 127.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagKey,
			err:   ``,
		},
	})
}

func TestAccTagValueAWS(t *testing.T) {
	testutil.Require(t, testutil.AWS)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "invalid",
			value: "café",
			fn:    testTagValue,
			err:   ` The value must match the regular expression ^[\d \w\+\-=\.:/@]*$.`,
		},
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagValue,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagValue,
			err:   ``,
		},
	})
}

func TestAccTagKeyAzure(t *testing.T) {
	testutil.Require(t, testutil.Azure)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "invalid",
			value: "café?",
			fn:    testTagKey,
			err:   ` The key must match the regular expression ^[^<>\*&%;\\\/\+\?]*$.`,
		},
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagKey,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "empty",
			value: "",
			fn:    testTagKey,
			err:   ` the minimal length is 1, and the maximum length is 512.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagKey,
			err:   ``,
		},
	})
}

func TestAccTagValueAzure(t *testing.T) {
	testutil.Require(t, testutil.Azure)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagValue,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagValue,
			err:   ``,
		},
	})
}

func TestAccTagKeyGCP(t *testing.T) {
	testutil.Require(t, testutil.GCP)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "invalid",
			value: "café?",
			fn:    testTagKey,
			err:   ` The key must match the regular expression ^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$.`,
		},
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagKey,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "empty",
			value: "",
			fn:    testTagKey,
			err:   ` the minimal length is 1, and the maximum length is 63.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagKey,
			err:   ``,
		},
	})
}

func TestAccTagValueGCP(t *testing.T) {
	testutil.Require(t, testutil.GCP)
	t.Parallel()

	runTagTestCases(t, []tagTestCase{
		{
			name:  "invalid",
			value: "café",
			fn:    testTagValue,
			err:   ` The value must match the regular expression ^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$.`,
		},
		{
			name:  "unicode",
			value: "🍎",
			fn:    testTagValue,
			err:   ` contains non-latin1 characters.`,
		},
		{
			name:  "valid",
			value: "cafe",
			fn:    testTagValue,
			err:   ``,
		},
	})
}
