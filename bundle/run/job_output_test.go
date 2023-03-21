package run

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleTaskJobOutputToString(t *testing.T) {
	taskNotebook := NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	myJob := JobOutput{
		RunPageUrl: "my_job_url",
		TaskOutputs: map[string]RunOutput{
			"my_notebook_task": &taskNotebook,
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
		RunPageUrl: "my_job_url",
		TaskOutputs: map[string]RunOutput{
			"my_foo_task": &taskFoo,
			"my_bar_task": &taskBar,
		},
	}

	actual, err := myJob.String()
	require.NoError(t, err)

	expected := `Run URL: my_job_url
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

func TestNotebookOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.RunOutput{
		NotebookOutput: &jobs.NotebookOutput{
			Result:    "foo",
			Truncated: true,
		},
		Logs:          "hello :)",
		LogsTruncated: true,
	}
	actual := toRunOutput(jobOutput)

	expected := &NotebookOutput{
		Result:    "foo",
		Truncated: true,
	}
	assert.Equal(t, expected, actual)
}

func TestDbtOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.RunOutput{
		DbtOutput: &jobs.DbtOutput{
			ArtifactsLink: "foo",
		},
		Logs: "hello :)",
	}
	actual := toRunOutput(jobOutput)

	expected := &DbtOutput{
		ArtifactsLink: "foo",
	}
	assert.Equal(t, expected, actual)
}

func TestSqlOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.RunOutput{
		SqlOutput: &jobs.SqlOutput{
			QueryOutput: &jobs.SqlQueryOutput{
				OutputLink: "foo",
			},
		},
		Logs: "hello :)",
	}
	actual := toRunOutput(jobOutput)

	expected := &SqlOutput{
		QueryOutput: &jobs.SqlQueryOutput{
			OutputLink: "foo",
		},
	}
	assert.Equal(t, expected, actual)
}

func TestLogOutputToRunOutput(t *testing.T) {
	jobOutput := &jobs.RunOutput{
		Logs:          "hello :)",
		LogsTruncated: true,
	}
	actual := toRunOutput(jobOutput)

	expected := &LogsOutput{
		Logs:          "hello :)",
		LogsTruncated: true,
	}
	assert.Equal(t, expected, actual)
}
