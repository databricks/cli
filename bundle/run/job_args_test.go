package run

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestJobParameterArgs(t *testing.T) {
	a := jobParameterArgs{
		&resources.Job{
			JobSettings: jobs.JobSettings{
				Parameters: []jobs.JobParameterDefinition{
					{
						Name:    "foo",
						Default: "value",
					},
					{
						Name:    "bar",
						Default: "value",
					},
				},
			},
		},
	}

	t.Run("ParseArgsError", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"--p1=v1", "superfluous"}, &opts)
		assert.ErrorContains(t, err, "unexpected positional arguments")
	})

	t.Run("ParseArgs", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"--p1=v1", "--p2=v2"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			map[string]string{
				"p1": "v1",
				"p2": "v2",
			},
			opts.Job.jobParams,
		)
	})

	t.Run("ParseArgsAppend", func(t *testing.T) {
		var opts Options
		opts.Job.jobParams = map[string]string{"p1": "v1"}
		err := a.ParseArgs([]string{"--p2=v2"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			map[string]string{
				"p1": "v1",
				"p2": "v2",
			},
			opts.Job.jobParams,
		)
	})

	t.Run("CompleteArgs", func(t *testing.T) {
		completions, _ := a.CompleteArgs([]string{}, "")
		assert.Equal(t, []string{"--foo=", "--bar="}, completions)
	})
}

func TestJobTaskNotebookParamArgs(t *testing.T) {
	a := jobTaskNotebookParamArgs{
		&resources.Job{
			JobSettings: jobs.JobSettings{
				Tasks: []jobs.Task{
					{
						NotebookTask: &jobs.NotebookTask{
							BaseParameters: map[string]string{
								"foo": "value",
								"bar": "value",
							},
						},
					},
				},
			},
		},
	}

	t.Run("ParseArgsError", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"--p1=v1", "superfluous"}, &opts)
		assert.ErrorContains(t, err, "unexpected positional arguments")
	})

	t.Run("ParseArgs", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"--p1=v1", "--p2=v2"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			map[string]string{
				"p1": "v1",
				"p2": "v2",
			},
			opts.Job.notebookParams,
		)
	})

	t.Run("ParseArgsAppend", func(t *testing.T) {
		var opts Options
		opts.Job.notebookParams = map[string]string{"p1": "v1"}
		err := a.ParseArgs([]string{"--p2=v2"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			map[string]string{
				"p1": "v1",
				"p2": "v2",
			},
			opts.Job.notebookParams,
		)
	})

	t.Run("CompleteArgs", func(t *testing.T) {
		completions, _ := a.CompleteArgs([]string{}, "")
		assert.ElementsMatch(t, []string{"--foo=", "--bar="}, completions)
	})
}

func TestJobTaskJarParamArgs(t *testing.T) {
	a := jobTaskJarParamArgs{}

	t.Run("ParseArgs", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"foo", "bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.jarParams,
		)
	})

	t.Run("ParseArgsAppend", func(t *testing.T) {
		var opts Options
		opts.Job.jarParams = []string{"foo"}
		err := a.ParseArgs([]string{"bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.jarParams,
		)
	})

	t.Run("CompleteArgs", func(t *testing.T) {
		completions, _ := a.CompleteArgs([]string{}, "")
		assert.Empty(t, completions)
	})
}

func TestJobTaskPythonParamArgs(t *testing.T) {
	a := jobTaskPythonParamArgs{}

	t.Run("ParseArgs", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"foo", "bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.pythonParams,
		)
	})

	t.Run("ParseArgsAppend", func(t *testing.T) {
		var opts Options
		opts.Job.pythonParams = []string{"foo"}
		err := a.ParseArgs([]string{"bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.pythonParams,
		)
	})

	t.Run("CompleteArgs", func(t *testing.T) {
		completions, _ := a.CompleteArgs([]string{}, "")
		assert.Empty(t, completions)
	})
}

func TestJobTaskSparkSubmitParamArgs(t *testing.T) {
	a := jobTaskSparkSubmitParamArgs{}

	t.Run("ParseArgs", func(t *testing.T) {
		var opts Options
		err := a.ParseArgs([]string{"foo", "bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.sparkSubmitParams,
		)
	})

	t.Run("ParseArgsAppend", func(t *testing.T) {
		var opts Options
		opts.Job.sparkSubmitParams = []string{"foo"}
		err := a.ParseArgs([]string{"bar"}, &opts)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]string{"foo", "bar"},
			opts.Job.sparkSubmitParams,
		)
	})

	t.Run("CompleteArgs", func(t *testing.T) {
		completions, _ := a.CompleteArgs([]string{}, "")
		assert.Empty(t, completions)
	})
}
