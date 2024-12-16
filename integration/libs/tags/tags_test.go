package tags_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func testTags(t *testing.T, tags map[string]string) error {
	ctx, wt := acc.WorkspaceTest(t)
	resp, err := wt.W.Jobs.Create(ctx, jobs.CreateJob{
		Name: testutil.RandomName("test-tags-"),
		Tasks: []jobs.Task{
			{
				TaskKey: "test",
				NewCluster: &compute.ClusterSpec{
					SparkVersion: "13.3.x-scala2.12",
					NumWorkers:   1,
					NodeTypeId:   testutil.GetCloud(t).NodeTypeID(),
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
			_ = wt.W.Jobs.DeleteByJobId(ctx, resp.JobId)
			// Cannot enable errchecking there, tests fail with:
			//   Error: Received unexpected error:
			//   Job 0 does not exist.
			// require.NoError(t, err)
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

func TestTagKeyAWS(t *testing.T) {
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

func TestTagValueAWS(t *testing.T) {
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

func TestTagKeyAzure(t *testing.T) {
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

func TestTagValueAzure(t *testing.T) {
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

func TestTagKeyGCP(t *testing.T) {
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

func TestTagValueGCP(t *testing.T) {
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
