package output

import (
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotebookOutputToString(t *testing.T) {
	taskFoo := NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	taskBar := NotebookOutput{
		Result:    "bar",
		Truncated: false,
	}

	actualFoo, err := taskFoo.String()
	require.NoError(t, err)
	assert.Equal(t, "foo\n[truncated...]\n", actualFoo)

	actualBar, err := taskBar.String()
	require.NoError(t, err)
	assert.Equal(t, "bar", actualBar)
}

func TestLogsOutputToString(t *testing.T) {
	taskFoo := LogsOutput{
		Logs:          "foo",
		LogsTruncated: true,
	}
	taskBar := LogsOutput{
		Logs:          "bar",
		LogsTruncated: false,
	}

	actualFoo, err := taskFoo.String()
	require.NoError(t, err)
	assert.Equal(t, "foo\n[truncated...]\n", actualFoo)

	actualBar, err := taskBar.String()
	require.NoError(t, err)
	assert.Equal(t, "bar", actualBar)
}

func TestDbtOutputToString(t *testing.T) {
	task := DbtOutput{
		ArtifactsHeaders: map[string]string{"a": "b", "c": "d"},
		ArtifactsLink:    "my_link",
	}

	actual, err := task.String()
	expected := `Dbt Task Output:
{
  "artifacts_headers": {
    "a": "b",
    "c": "d"
  },
  "artifacts_link": "my_link"
}`
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSqlOutputToString(t *testing.T) {
	task := SqlOutput{
		QueryOutput: &jobs.SqlQueryOutput{
			OutputLink:  "a",
			QueryText:   "b",
			WarehouseId: "d",
		},
	}

	actual, err := task.String()
	expected := `SQL Task Output:
{
  "query_output": {
    "output_link": "a",
    "query_text": "b",
    "warehouse_id": "d"
  }
}`
	require.NoError(t, err)
	fmt.Println("[DEBUG] actual: ", actual)
	assert.Equal(t, expected, actual)
}
