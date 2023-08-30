package python

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Actual   []string
	Expected string
}
type NamedParams map[string]string
type testCaseNamed struct {
	Actual   NamedParams
	Expected string
}

var paramsTestCases []testCase = []testCase{
	{[]string{}, `"python"`},
	{[]string{"a"}, `"python", "a"`},
	{[]string{"a", "b"}, `"python", "a", "b"`},
	{[]string{"123!@#$%^&*()-="}, `"python", "123!@#$%^&*()-="`},
	{[]string{`{"a": 1}`}, `"python", "{\"a\": 1}"`},
}

var paramsTestCasesNamed []testCaseNamed = []testCaseNamed{
	{NamedParams{}, `"python"`},
	{NamedParams{"a": "1"}, `"python", "a=1"`},
	{NamedParams{"a": "'1'"}, `"python", "a='1'"`},
	{NamedParams{"a": `"1"`}, `"python", "a=\"1\""`},
	{NamedParams{"a": "1", "b": "2"}, `"python", "a=1", "b=2"`},
	{NamedParams{"data": `{"a": 1}`}, `"python", "data={\"a\": 1}"`},
}

func TestGenerateParameters(t *testing.T) {
	trampoline := pythonTrampoline{}
	for _, c := range paramsTestCases {
		task := &jobs.PythonWheelTask{Parameters: c.Actual}
		result, err := trampoline.generateParameters(task)
		require.NoError(t, err)
		require.Equal(t, c.Expected, result)
	}
}

func TestGenerateNamedParameters(t *testing.T) {
	trampoline := pythonTrampoline{}
	for _, c := range paramsTestCasesNamed {
		task := &jobs.PythonWheelTask{NamedParameters: c.Actual}
		result, err := trampoline.generateParameters(task)
		require.NoError(t, err)

		// parameters order can be undetermenistic, so just check that they exist as expected
		require.ElementsMatch(t, strings.Split(c.Expected, ","), strings.Split(result, ","))
	}
}

func TestGenerateBoth(t *testing.T) {
	trampoline := pythonTrampoline{}
	task := &jobs.PythonWheelTask{NamedParameters: map[string]string{"a": "1"}, Parameters: []string{"b"}}
	_, err := trampoline.generateParameters(task)
	require.Error(t, err)
	require.ErrorContains(t, err, "not allowed to pass both paramaters and named_parameters")
}

func TestNoPanicWithNoPythonWheelTasks(t *testing.T) {
	tmpDir := t.TempDir()
	b := &bundle.Bundle{
		Config: config.Root{
			Path: tmpDir,
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						Paths: resources.Paths{
							ConfigFilePath: tmpDir,
						},
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:      "notebook_task",
									NotebookTask: &jobs.NotebookTask{}},
							},
						},
					},
				},
			},
		},
	}
	trampoline := TransformWheelTask()
	err := bundle.Apply(context.Background(), b, trampoline)
	require.NoError(t, err)
}
