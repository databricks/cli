package python

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Actual   []string
	Expected string
}

type testCaseNamed struct {
	Actual   map[string]string
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
	{map[string]string{}, `"python"`},
	{map[string]string{"a": "1"}, `"python", "a=1"`},
	{map[string]string{"a": "'1'"}, `"python", "a='1'"`},
	{map[string]string{"a": `"1"`}, `"python", "a=\"1\""`},
	{map[string]string{"a": "1", "b": "2"}, `"python", "a=1", "b=2"`},
	{map[string]string{"data": `{"a": 1}`}, `"python", "data={\"a\": 1}"`},
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

func TestTransformFiltersWheelTasksOnly(t *testing.T) {
	trampoline := pythonTrampoline{}
	bundle := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/test@test.com/bundle/dist/test.whl"},
									},
								},
								{
									TaskKey:      "key2",
									NotebookTask: &jobs.NotebookTask{},
								},
								{
									TaskKey:         "key3",
									PythonWheelTask: &jobs.PythonWheelTask{},
									Libraries: []compute.Library{
										{Whl: "dbfs:/FileStore/dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tasks := trampoline.GetTasks(bundle)
	require.Len(t, tasks, 1)
	require.Equal(t, "job1", tasks[0].JobKey)
	require.Equal(t, "key1", tasks[0].Task.TaskKey)
	require.NotNil(t, tasks[0].Task.PythonWheelTask)
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
						Paths: paths.Paths{
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
