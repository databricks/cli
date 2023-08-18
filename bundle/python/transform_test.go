package python

import (
	"testing"

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
	{NamedParams{"a": "1", "b": "2"}, `"python", "a=1", "b=2"`},
	{NamedParams{"data": `{"a": 1}`}, `"python", "data={\"a\": 1}"`},
}

func TestGenerateParameters(t *testing.T) {
	for _, c := range paramsTestCases {
		task := &jobs.PythonWheelTask{Parameters: c.Actual}
		result, err := generateParameters(task)
		require.NoError(t, err)
		require.Equal(t, c.Expected, result)
	}
}

func TestGenerateNamedParameters(t *testing.T) {
	for _, c := range paramsTestCasesNamed {
		task := &jobs.PythonWheelTask{NamedParameters: c.Actual}
		result, err := generateParameters(task)
		require.NoError(t, err)
		require.Equal(t, c.Expected, result)
	}
}

func TestGenerateBoth(t *testing.T) {
	task := &jobs.PythonWheelTask{NamedParameters: map[string]string{"a": "1"}, Parameters: []string{"b"}}
	_, err := generateParameters(task)
	require.Error(t, err)
}
