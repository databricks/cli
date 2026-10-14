package output

import (
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleTaskJobOutputToString(t *testing.T) {
	taskNotebook := NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	myJob := JobOutput{
		TaskOutputs: []TaskOutput{
			{TaskKey: "my_notebook_task", Output: &taskNotebook, EndTime: 0},
		},
	}

	actual, err := myJob.String()
	require.NoError(t, err)
	expected := "foo\n[truncated...]\n"
	assert.Equal(t, expected, actual)
}

func TestMultiTaskJobOutputToString(t *testing.T) {
	taskFoo := NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	taskBar := LogsOutput{
		Logs:          "bar",
		LogsTruncated: false,
	}
	myJob := JobOutput{
		TaskOutputs: []TaskOutput{
			{TaskKey: "my_bar_task", Output: &taskBar, EndTime: 0},
			{TaskKey: "my_foo_task", Output: &taskFoo, EndTime: 0},
		},
	}

	actual, err := myJob.String()
	require.NoError(t, err)

	expected := `Output:
=======
Task my_bar_task:
bar
=======
Task my_foo_task:
foo
[truncated...]

`
	assert.Equal(t, expected, actual)
}

func TestTaskJobOutputOrderToString(t *testing.T) {
	taskFoo := NotebookOutput{
		Result: "foo",
	}
	taskBar := LogsOutput{
		Logs: "bar",
	}
	taskBaz := LogsOutput{
		Logs: "baz",
	}
	myJob := JobOutput{
		TaskOutputs: []TaskOutput{
			{TaskKey: "my_baz_task", Output: &taskBaz, EndTime: 1683553233331},
			{TaskKey: "my_bar_task", Output: &taskBar, EndTime: 1683553223508},
			{TaskKey: "my_foo_task", Output: &taskFoo, EndTime: 1683553217598},
		},
	}

	actual, err := myJob.String()
	require.NoError(t, err)

	expected := `Output:
=======
Task my_foo_task:
foo
=======
Task my_bar_task:
bar
=======
Task my_baz_task:
baz
`
	assert.Equal(t, expected, actual)
}

func TestNotebookOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.GetRunOutputResponse{
		NotebookOutput: &jobs.NotebookTask_NotebookOutput{
			Result:    new("foo"),
			Truncated: new(true),
		},
		Logs:          new("hello :)"),
		LogsTruncated: new(true),
	}
	actual := toRunOutput(jobOutput)

	expected := &NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	assert.Equal(t, expected, actual)
}

func TestNotebookOutputWithEmptyResultFallsBackToLogs(t *testing.T) {
	jobOutput := &jobs.GetRunOutputResponse{
		NotebookOutput: &jobs.NotebookTask_NotebookOutput{
			Result: new(""),
		},
		Logs:          new("hello :)"),
		LogsTruncated: new(true),
	}
	actual := toRunOutput(jobOutput)

	expected := &LogsOutput{
		Logs:          "hello :)",
		LogsTruncated: true,
	}
	assert.Equal(t, expected, actual)
}

func TestDbtOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.GetRunOutputResponse{
		DbtOutput: &jobs.DbtTask_DbtTaskOutput{
			ArtifactsLink: new("foo"),
		},
		Logs: new("hello :)"),
	}
	actual := toRunOutput(jobOutput)

	expected := &DbtOutput{
		ArtifactsLink: "foo",
	}
	assert.Equal(t, expected, actual)
}

func TestSqlOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.GetRunOutputResponse{
		SqlOutput: &jobs.SqlTask_SqlOutput{
			QueryOutput: &jobs.SqlTask_SqlQueryOutput{
				OutputLink: new("foo"),
			},
		},
		Logs: new("hello :)"),
	}
	actual := toRunOutput(jobOutput)

	expected := &SqlOutput{
		QueryOutput: &jobs.SqlTask_SqlQueryOutput{
			OutputLink: new("foo"),
		},
	}
	assert.Equal(t, expected, actual)
}

func TestLogOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.GetRunOutputResponse{
		Logs:          new("hello :)"),
		LogsTruncated: new(true),
	}
	actual := toRunOutput(jobOutput)

	expected := &LogsOutput{
		Logs:          "hello :)",
		LogsTruncated: true,
	}
	assert.Equal(t, expected, actual)
}
