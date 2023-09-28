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

func TestAccTagKeyAWS(t *testing.T) {
	testutil.Require(t, testutil.AWS)
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "café")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, `The key must match the regular expression ^[\d \w\+\-=\.:\/@]*$.`)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, ` the minimal length is 1, and the maximum length is 127.`)
	})
}

func TestAccTagValueAWS(t *testing.T) {
	testutil.Require(t, testutil.AWS)
	t.Parallel()

	err := testTagValue(t, "café")
	require.Error(t, err)
	msg := strings.ReplaceAll(err.Error(), "\n", " ")
	require.Contains(t, msg, `The value must match the regular expression ^[\d \w\+\-=\.:/@]*$.`)
}

func TestAccTagKeyAzure(t *testing.T) {
	testutil.Require(t, testutil.Azure)
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "café?")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, `The key must match the regular expression ^[^<>\*&%;\\\/\+\?]*$.`)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, ` the minimal length is 1, and the maximum length is 512.`)
	})
}

func TestAccTagValueAzure(t *testing.T) {
	testutil.Require(t, testutil.Azure)
	t.Parallel()

	// Azure puts no constraings on tag values.
	err := testTagValue(t, "café?")
	require.NoError(t, err)
}

func TestAccTagKeyGCP(t *testing.T) {
	testutil.Require(t, testutil.GCP)
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "café?")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, `The key must match the regular expression ^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$.`)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		err := testTagKey(t, "")
		require.Error(t, err)
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		require.Contains(t, msg, ` the minimal length is 1, and the maximum length is 63.`)
	})
}

func TestAccTagValueGCP(t *testing.T) {
	testutil.Require(t, testutil.GCP)
	t.Parallel()

	err := testTagValue(t, "café?")
	require.Error(t, err)
	msg := strings.ReplaceAll(err.Error(), "\n", " ")
	require.Contains(t, msg, `The value must match the regular expression ^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$.`)
}
